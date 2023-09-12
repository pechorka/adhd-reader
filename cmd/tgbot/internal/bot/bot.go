package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/pechorka/adhd-reader/internal/service"
	"github.com/pechorka/adhd-reader/internal/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pechorka/adhd-reader/pkg/contenttype"
	"github.com/pechorka/adhd-reader/pkg/filechecksum"
	"github.com/pechorka/adhd-reader/pkg/fileloader"
	"github.com/pechorka/adhd-reader/pkg/i18n"
	"github.com/pechorka/adhd-reader/pkg/pdfexctractor"
	"github.com/pechorka/adhd-reader/pkg/queue"
	"github.com/pechorka/adhd-reader/pkg/runeslice"
	"github.com/pechorka/adhd-reader/pkg/sizeconverter"
)

const (
	textSelect = "text-select:"
	deleteText = "delete-text:"
	nextChunk  = "next-chunk"
	prevChunk  = "prev-chunk"
	rereadText = "reread-text:"
	nextPage   = "next-page:"
)

const (
	defaultMaxFileSize = 20 * 1024 * 1024 // 20 MB
	defaultPageSize    = 50
)

type Bot struct {
	service     *service.Service
	bot         *tgbotapi.BotAPI
	msgQueue    *queue.MessageQueue
	fileLoader  *fileloader.Loader
	i18n        *i18n.Localies
	maxFileSize int
	adminUsers  map[int64]struct{}
}

type Config struct {
	Token       string
	Service     *service.Service
	MsgQueue    *queue.MessageQueue
	FileLoader  *fileloader.Loader
	I18n        *i18n.Localies
	MaxFileSize int
	AdminUsers  []int64
}

func NewBot(cfg Config) (*Bot, error) {
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = defaultMaxFileSize
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	bot.Debug = true // TODO before release take from config

	adminUsers := make(map[int64]struct{}, len(cfg.AdminUsers))
	for _, id := range cfg.AdminUsers {
		adminUsers[id] = struct{}{}
	}
	return &Bot{
		service:     cfg.Service,
		bot:         bot,
		msgQueue:    cfg.MsgQueue,
		fileLoader:  cfg.FileLoader,
		i18n:        cfg.I18n,
		maxFileSize: cfg.MaxFileSize,
		adminUsers:  adminUsers,
	}, nil
}

func validateConfig(cfg Config) error {
	if cfg.Token == "" {
		return fmt.Errorf("token is empty")
	}
	if cfg.Service == nil {
		return fmt.Errorf("service is nil")
	}
	if cfg.MsgQueue == nil {
		return fmt.Errorf("msgQueue is nil")
	}
	if cfg.FileLoader == nil {
		return fmt.Errorf("fileLoader is nil")
	}
	if cfg.I18n == nil {
		return fmt.Errorf("i18n is nil")
	}
	return nil
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	b.msgQueue.Run(b.onQueueFilled)
	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		if msg := update.Message; msg != nil {
			b.handleMsg(msg)
		}

		if cb := update.CallbackQuery; cb != nil {
			b.handleCallback(cb)
		}
	}
}

func (b *Bot) Stop() {
	b.bot.StopReceivingUpdates()
	b.msgQueue.Stop()
}

func (b *Bot) handlePanic(user *tgbotapi.User) {
	if rec := recover(); rec != nil {
		b.replyToUserWithI18n(user, panicMsgId)
		b.reportError(fmt.Sprintf("Я запаниковал: %v", rec))
		log.Println("Panic: ", rec, "Stack: ", string(debug.Stack()))
	}
}

func (b *Bot) handleMsg(msg *tgbotapi.Message) {
	defer b.handlePanic(msg.From)

	if msg.Document != nil {
		b.saveTextFromDocument(msg)
		return
	}

	switch cmd := msg.Command(); cmd {
	case "start":
		b.start(msg)
	case "list":
		b.listCmd(msg)
	case "page":
		b.onPageCommand(msg)
	case "chunk":
		b.chunk(msg)
	case "delete":
		b.delete(msg)
	case "rename":
		b.rename(msg)
	case "download":
		b.download(msg)
	case "help":
		b.help(msg)
	case "random":
		b.random(msg, -1)
	case "random50":
		b.random(msg, 50)
	case "loot":
		b.loot(msg)
	case "stats":
		b.stats(msg)
	default:
		if cmd != "" {
			if b.handleAdminMsg(msg) {
				return
			}
			log.Println("Unknown command: ", cmd)
			b.replyToUserWithI18n(msg.From, errorUnknownCommandMsgId)
			return
		}
		b.saveTextFromMessage(msg)
	}

	// command for bot father to add command help
	/*
		/setcommands
		list - list all texts
		page - set page number, pass page number as argument
		chunk - set chunk size, pass chunk size as argument
		delete - delete text, pass text name as argument
		rename - rename text, pass new name as argument
		download - download all texts in json format
		help - troubleshooting and support
	*/
}

func (b *Bot) handleAdminMsg(msg *tgbotapi.Message) bool {
	if _, ok := b.adminUsers[msg.From.ID]; !ok {
		return false
	}
	switch cmd := msg.Command(); cmd {
	case "analytics":
		b.analytics(msg)
	default:
		return false
	}

	return true
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	defer b.handlePanic(cb.From)

	switch {
	case strings.HasPrefix(cb.Data, textSelect):
		b.selectTextCallBack(cb)
	case strings.HasPrefix(cb.Data, deleteText):
		b.deleteTextCallBack(cb)
	case cb.Data == nextChunk:
		b.nextChunk(cb.From)
	case cb.Data == prevChunk:
		b.prevChunk(cb.From)
	case strings.HasPrefix(cb.Data, rereadText):
		b.rereadText(cb)
	case strings.HasPrefix(cb.Data, nextPage):
		b.nextListPage(cb)
	}
	// Respond to the callback query, telling Telegram to show the user
	// a message with the data received.
	callback := tgbotapi.NewCallback(cb.ID, cb.Data)
	if _, err := b.bot.Request(callback); err != nil {
		log.Println("Failed to respond to callback query:", err)
	}
}

func (b *Bot) selectTextCallBack(cb *tgbotapi.CallbackQuery) {
	textUUID := strings.TrimPrefix(cb.Data, textSelect)
	b.selectText(cb.From, textUUID)
}

func (b *Bot) selectText(from *tgbotapi.User, textUUID string) {
	currentText, err := b.service.SelectText(from.ID, textUUID)
	if err != nil {
		b.replyErrorToUserWithI18n(from, errorOnTextSelectMsgId, err)
		return
	}
	b.replyToUserWithI18nWithArgs(from, onTextSelectMsgId, map[string]string{
		"text_name": currentText.Name,
	})
	b.currentChunk(from)
}

func (b *Bot) deleteTextCallBack(cb *tgbotapi.CallbackQuery) {
	textUUID := strings.TrimPrefix(cb.Data, deleteText)
	err := b.service.DeleteTextByUUID(cb.From.ID, textUUID)
	if err != nil {
		b.replyErrorToUserWithI18n(cb.From, errorOnTextDeleteMsgId, err)
		return
	}
	b.replyToUserWithI18n(cb.From, onTextDeletedMsgId)
}

func (b *Bot) nextChunk(from *tgbotapi.User) {
	loot, err := b.service.LootOnNextChunk(from.ID)
	if err != nil {
		b.replyErrorToUser(from, errorOnGettingLootMsgId, err)
	} else {
		if loot.DeltaDust.TotalDust() > 0 || loot.DeltaHerb.TotalHerb() > 0 {
			if loot.DeltaDust.TotalDust() > 0 && loot.DeltaHerb.TotalHerb() > 0 {
				b.replyWithPlainText(from, DustToString(loot.DeltaDust, " ")+HerbToString(loot.DeltaHerb, " "))
			} else {
				if loot.DeltaDust.TotalDust() > 0 {
					b.replyWithPlainText(from, DustToString(loot.DeltaDust, " "))
				} else {
					b.replyWithPlainText(from, HerbToString(loot.DeltaHerb, " "))
				}
			}

		}
	}
	lvl, _, levelUp, err := b.service.ExpOnNextChunk(from.ID)
	// b.replyWithPlainText(from, "EXP:"+strconv.FormatInt(lvl.Experience, 10)+"LVL:"+strconv.FormatInt(lvl.Level, 10))
	if err != nil {
		b.replyErrorToUser(from, errorOnGettingLootMsgId, err)
	}
	if levelUp {
		b.replyWithPlainText(from, "🎉 LEVEL UP!! Level "+strconv.FormatInt(lvl.Level, 10)) //TODO: i18n перевести
		//alert me if someone reached level 10+ so that i can refactor leveling system
		if lvl.Level > 10 {
			b.reportError("user " + strconv.FormatInt(from.ID, 10) + " reached level " + strconv.FormatInt(lvl.Level, 10) + "!")
		}
	}

	b.chunkReply(from, b.service.NextChunk)
}

func (b *Bot) prevChunk(from *tgbotapi.User) {
	b.chunkReply(from, b.service.PrevChunk)
}

func (b *Bot) currentChunk(from *tgbotapi.User) {
	b.chunkReply(from, b.service.CurrentOrFirstChunk)
}

func (b *Bot) rereadText(cb *tgbotapi.CallbackQuery) {
	textUUID := strings.TrimPrefix(cb.Data, rereadText)
	_, err := b.service.SelectText(cb.From.ID, textUUID)
	if err != nil {
		b.replyErrorToUserWithI18n(cb.From, errorOnTextSelectMsgId, err)
		return
	}
	b.setPage(cb.From, 0)
}

func (b *Bot) nextListPage(cb *tgbotapi.CallbackQuery) {
	page, err := strconv.Atoi(strings.TrimPrefix(cb.Data, nextPage))
	if err != nil {
		b.replyErrorToUserWithI18n(cb.From, errorOnParsingListPageMsgId, err)
		return
	}
	b.list(cb.From, page, defaultPageSize)
}

type chunkSelectorFunc func(userID int64) (storage.Text, string, service.ChunkType, error)

func (b *Bot) chunkReply(from *tgbotapi.User, chunkSelector chunkSelectorFunc) {
	currentText, chunkText, chunkType, err := chunkSelector(from.ID)
	prevBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(from, previousButtonMsgId), prevChunk)
	nextBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(from, nextButtonMsgId), nextChunk)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(from, deleteButtonMsgId), deleteText+currentText.UUID)
	rereadBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(from, rereadButtonMsgId), rereadText+currentText.UUID)
	// #29 TODO code for reread button
	switch err {
	case service.ErrFirstChunk:
		b.replyToUserWithI18n(from, warningFirstChunkCantGoBackMsgId, nextBtn)
		return
	case service.ErrTextFinished:
		b.replyToUserWithI18nWithArgs(from, textFinishedMsgId, map[string]string{
			"text_name": currentText.Name,
		}, prevBtn, deleteBtn)
	case nil:
	default:
		b.replyErrorToUserWithI18n(from, erroroOnGettingNextChunk, err)
		return
	}

	if chunkText == "" && chunkType != service.ChunkTypeLast {
		b.replyToUserWithI18nWithArgs(from, errorEmptyChunkMsgId, map[string]string{
			"text_name": currentText.Name,
		}, deleteBtn)
		return
	}

	switch chunkType {
	case service.ChunkTypeFirst:
		b.replyWithPlainText(from, chunkText, nextBtn)
	case service.ChunkTypeLast:
		b.replyToUserWithI18nWithArgs(from, lastChunkMsgId, map[string]string{
			"text_name": currentText.Name,
		}, prevBtn, deleteBtn, rereadBtn)
	default:
		b.replyWithPlainText(from, chunkText, prevBtn, nextBtn)
	}
}

func (b *Bot) start(msg *tgbotapi.Message) {
	go func() { // todo: stop flow on other commands???
		b.replyToUserWithI18n(msg.From, firstMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.replyToUserWithI18n(msg.From, secondMsg)
		b.sendTyping(msg)
		time.Sleep(5 * time.Second)

		b.replyToUserWithI18n(msg.From, thirdMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.replyToUserWithI18n(msg.From, fourthMsg)
		b.replyToUserWithI18n(msg.From, fifthMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.replyToUserWithI18n(msg.From, sixthMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		file := startFileEn
		fileName := startFileNameEn
		if getLanguageCode(msg.From) == langCodeRu {
			file = startFileRu
			fileName = startFileNameRu
		}
		fileMsg := tgbotapi.NewDocument(msg.From.ID, tgbotapi.FileBytes{
			Name:  fileName,
			Bytes: file,
		})
		b.send(fileMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.replyToUserWithI18n(msg.From, eighthMsg)
	}()
}

func (b *Bot) listCmd(msg *tgbotapi.Message) {
	b.list(msg.From, 1, defaultPageSize)
}

func (b *Bot) list(from *tgbotapi.User, page, pageSize int) {
	texts, more, err := b.service.ListTexts(from.ID, page, pageSize)
	if err != nil {
		b.replyErrorToUserWithI18n(from, errorOnListMsgId, err)
		return
	}
	if len(texts) == 0 {
		b.replyToUserWithI18n(from, warningNoTextsMsgId)
		return
	}
	// reply with button for each text and save text index in callback data
	var buttons []tgbotapi.InlineKeyboardButton
	for _, t := range texts {
		btnText := completionPercentString(t.CompletionPercent) + " " + t.Name
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(btnText, textSelect+t.UUID))
	}
	if more {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(b.getText(from, nextButtonMsgId), nextPage+strconv.Itoa(page+1)))
	}
	b.replyToUserWithI18n(from, onListMsgId, buttons...)
}

func completionPercentString(percent int) string {
	switch percent {
	case 0:
		return "🆕"
	case 100:
		return "✅"
	default:
		return fmt.Sprintf("(%d%%)", percent)
	}
}

func (b *Bot) onPageCommand(msg *tgbotapi.Message) {
	strPage := msg.CommandArguments()
	page, err := strconv.ParseInt(strings.TrimSpace(strPage), 10, 64)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnParsingPageMsgId, err)
		return
	}
	b.setPage(msg.From, page)
}

func (b *Bot) setPage(from *tgbotapi.User, page int64) {
	err := b.service.SetPage(from.ID, page)
	if err != nil {
		if err == service.ErrTextNotSelected {
			b.replyToUserWithI18n(from, errorOnSettingPageNoTextSelectedMsgId)
			return
		}
		b.replyErrorToUserWithI18n(from, errorOnSettingPageMsgId, err)
		return
	}
	b.replyToUserWithI18n(from, pageSetMsgId)
	b.currentChunk(from)
}

func (b *Bot) chunk(msg *tgbotapi.Message) {
	strChunk := msg.CommandArguments()
	chunk, err := strconv.ParseInt(strings.TrimSpace(strChunk), 10, 64)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnParsingChunkSizeMsgId, err)
		return
	}
	err = b.service.SetChunkSize(msg.From.ID, chunk)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnSettingChunkSizeMsgId, err)
		return
	}
	b.replyToMsgWithI18n(msg, chunkSizeSetMsgId)
}

func (b *Bot) delete(msg *tgbotapi.Message) {
	textName := strings.TrimSpace(msg.CommandArguments())
	err := b.service.DeleteTextByName(msg.From.ID, textName)
	if err != nil {
		b.replyErrorWithI18n(msg, onTextDeletedMsgId, err)
		return
	}
	b.replyToMsgWithI18n(msg, onTextDeletedMsgId)
}

func (b *Bot) rename(msg *tgbotapi.Message) {
	newName := strings.TrimSpace(msg.CommandArguments())
	oldName, err := b.service.RenameText(msg.From.ID, newName)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnTextRenameMsgId, err)
		return
	}
	b.replyToMsgWithI18nWithArgs(msg, onTextRenamedMsgId, map[string]string{
		"text_name":     oldName,
		"new_text_name": newName,
	})
}

func (b *Bot) download(msg *tgbotapi.Message) {
	texts, err := b.service.FullTexts(msg.From.ID)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnListMsgId, err)
		return
	}

	type OutputText struct {
		TextName     string   `json:"textName"`
		CurrentChunk int64    `json:"currentChunk"`
		Chunks       []string `json:"chunks"`
	}

	type Output struct {
		Texts []OutputText `json:"texts"`
	}

	outTexts := make([]OutputText, 0, len(texts))
	for _, t := range texts {
		outTexts = append(outTexts, OutputText{
			TextName:     t.Name,
			CurrentChunk: t.CurrentChunk,
			Chunks:       t.Chunks,
		})
	}

	out := Output{
		Texts: outTexts,
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnFullTextEncodeMsgId, err)
		return
	}

	doc := tgbotapi.FileBytes{Name: "all_texts.json", Bytes: outBytes}
	b.send(tgbotapi.NewDocument(msg.Chat.ID, doc))
}

func (b *Bot) help(msg *tgbotapi.Message) {
	b.replyToMsgWithI18n(msg, helpMsg)
}

func (b *Bot) random(msg *tgbotapi.Message, atMostChunks int64) {
	text, err := b.service.RandomText(msg.From.ID, atMostChunks)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnRandomTextMsgId, err)
		return
	}
	b.selectText(msg.From, text.UUID)
}

func (b *Bot) loot(msg *tgbotapi.Message) {
	dust, herb, err := b.service.GetLoot(msg.From.ID)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnGettingLootMsgId, err)
	}
	b.replyWithPlainText(msg.From, DustToString(dust, "\n")+"\n"+HerbToString(herb, "\n")+"\n ✨✨✨ "+strconv.FormatInt(dust.TotalDust(), 10)+"\n 🪴🪴🪴 "+strconv.FormatInt(herb.TotalHerb(), 10))
}

func (b *Bot) stats(msg *tgbotapi.Message) {

	stats, level, err := b.service.GetStatsAndLevel(msg.From.ID)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnGettingLootMsgId, err)
	}
	b.replyWithPlainText(msg.From, "LEVEL "+strconv.FormatInt(level.Level, 10)+"\n"+
		"EXP: "+strconv.FormatInt(level.Experience, 10)+"\n"+"✨✨✨✨✨"+"\n"+
		"AC: "+strconv.FormatInt(stats.Accuracy, 10)+"\n"+
		"AT: "+strconv.FormatInt(stats.Attention, 10)+"\n"+
		"TM: "+strconv.FormatInt(stats.TimeManagement, 10)+"\n"+
		"CR: "+strconv.FormatInt(stats.Charizma, 10)+"\n"+
		"LK: "+strconv.FormatInt(stats.Luck, 10)+"\n"+
		"Free: "+strconv.FormatInt(stats.Free, 10))

}

func (b *Bot) saveTextFromDocument(msg *tgbotapi.Message) {
	if msg.Document.FileSize != 0 && msg.Document.FileSize > b.maxFileSize {
		b.replyToMsgWithI18nWithArgs(msg, errorOnFileUploadTooBigMsgId, map[string]string{
			"max_file_size": sizeconverter.HumanReadableSizeInMB(b.maxFileSize),
		})
		return
	}
	switch {
	case contenttype.IsPlainText(msg.Document.MimeType):
	case contenttype.IsPDF(msg.Document.MimeType):
	default:
		b.replyToMsgWithI18n(msg, errorOnFileUploadInvalidFormatMsgId)
	}
	fileURL, err := b.bot.GetFileDirectURL(msg.Document.FileID)
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnFileUploadBuildingFileURLMsgId, err)
		return
	}
	data, err := b.fileLoader.DownloadFile(fileURL)
	switch err {
	case nil:
	case fileloader.ErrFileIsTooBig:
		b.replyToMsgWithI18nWithArgs(msg, errorOnFileUploadTooBigMsgId, map[string]string{
			"max_file_size": sizeconverter.HumanReadableSizeInMB(b.maxFileSize),
		})
		return
	default:
		b.replyErrorWithI18n(msg, errorOnFileUploadMsgId, err)
		return
	}

	var text string
	switch {
	case contenttype.IsPlainText(msg.Document.MimeType):
		text = string(data)
	case contenttype.IsPDF(msg.Document.MimeType):
		text, err = pdfexctractor.ExtractPlainTextFromPDF(data)
	}
	if err != nil {
		b.replyErrorWithI18n(msg, errorOnFileUploadExtractingTextMsgId, err)
		return
	}

	textID, err := b.service.AddTextFromFile(
		msg.From.ID,
		filechecksum.Calculate(data),
		msg.Document.FileName, text,
	)
	if err != nil {
		if err == service.ErrTextNotUTF8 {
			b.replyToMsgWithI18n(msg, errorOnTextSaveNotUTF8MsgId)
			return
		}
		var alreadyExists *storage.TextAlreadyExistsError
		if errors.As(err, &alreadyExists) {
			b.replyToMsgWithI18nWithArgs(msg, errorOnTextSaveAlreadyExistsMsgId, map[string]string{
				"text_name": alreadyExists.ExistingText.Name,
			})
			return
		}
		b.replyErrorWithI18n(msg, errorOnTextSaveMsgId, err)
		return
	}
	readBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(msg.From, readButtonMsgId), textSelect+textID)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(msg.From, deleteButtonMsgId), deleteText+textID)
	b.replyToMsgWithI18nWithArgs(msg, textSavedMsgId, map[string]string{
		"text_name": msg.Document.FileName,
	}, readBtn, deleteBtn)
}

func (b *Bot) saveTextFromMessage(msg *tgbotapi.Message) {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	if contenttype.IsURLs(text) {
		for _, link := range strings.Split(text, "\n") {
			textID, textName, err := b.service.AddTextFromURL(msg.From.ID, link)
			if err != nil {
				b.replyErrorWithI18n(msg, errorOnTextSaveMsgId, err)
				return
			}
			readBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(msg.From, readButtonMsgId), textSelect+textID)
			deleteBtn := tgbotapi.NewInlineKeyboardButtonData(b.getText(msg.From, deleteButtonMsgId), deleteText+textID)
			b.replyToMsgWithI18nWithArgs(msg, textSavedMsgId, map[string]string{
				"text_name": textName,
			}, readBtn, deleteBtn)
		}
		return
	}
	// else assume it's plain text
	b.msgQueue.Add(msg.From.ID, text)
}

func (b *Bot) onQueueFilled(userID int64, msgText string) {
	//@pechorka, не получилось, потому что там надо получать language code, а по юзер ИД такое нельзя сделать
	textName, _, ok := strings.Cut(msgText, "\n")
	const maxTextNameLength = 64
	if !ok || utf8.RuneCountInString(textName) > maxTextNameLength {
		textName = strings.TrimSpace(runeslice.NRunes(msgText, maxTextNameLength))
	}

	textID, err := b.service.AddText(userID, textName, msgText)
	if err != nil {
		b.sendToUser(userID, "Failed to save text: "+err.Error())
		return
	}
	readBtn := tgbotapi.NewInlineKeyboardButtonData("Read", textSelect+textID)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("Delete", deleteText+textID)
	b.sendToUser(userID, fmt.Sprintf("Text <code>%s</code> is saved", textName), readBtn, deleteBtn)
}

func (b *Bot) replyWithText(to *tgbotapi.Message, text string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	return b.sendMsg(msg, buttons...)
}

func (b *Bot) replyWithPlainText(to *tgbotapi.User, text string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(to.ID, text)
	return b.sendPlainTextMsg(msg, buttons...)
}

func (b *Bot) replyErrorWithI18n(msg *tgbotapi.Message, id string, err error, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	return b.replyError(msg, b.getTextWithArgs(msg.From, id, nil), err, buttons...)
}

func (b *Bot) replyErrorToUserWithI18n(from *tgbotapi.User, id string, err error, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	return b.replyErrorToUser(from, b.getTextWithArgs(from, id, nil), err, buttons...)
}

func (b *Bot) replyToMsgWithI18n(msg *tgbotapi.Message, id string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	return b.replyToMsgWithI18nWithArgs(msg, id, nil, buttons...)
}

func (b *Bot) replyToMsgWithI18nWithArgs(msg *tgbotapi.Message, id string, args map[string]string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	return b.replyWithText(msg, b.getTextWithArgs(msg.From, id, args), buttons...)
}

func (b *Bot) replyToUserWithI18n(from *tgbotapi.User, id string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	return b.replyToUserWithI18nWithArgs(from, id, nil, buttons...)
}

func (b *Bot) replyToUserWithI18nWithArgs(from *tgbotapi.User, id string, args map[string]string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	return b.sendToUser(from.ID, b.getTextWithArgs(from, id, args), buttons...)
}

func (b *Bot) getText(from *tgbotapi.User, textID string) string {
	return b.getTextWithArgs(from, textID, nil)
}

func (b *Bot) getTextWithArgs(from *tgbotapi.User, textID string, args map[string]string) string {
	langCode := getLanguageCode(from)
	var (
		text string
		err  error
	)
	if len(args) == 0 {
		text, err = b.i18n.Get(langCode, textID)
	} else {
		text, err = b.i18n.GetWithArgs(langCode, textID, args)
	}
	if err != nil {
		b.reportError(fmt.Sprintf("failed to get i18n text for id %s, locale %s: %v", textID, langCode, err))
		text = "Something went wrong"
	}
	return text
}

func (b *Bot) replyError(to *tgbotapi.Message, text string, err error, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(to.Chat.ID, text+": "+err.Error())
	msg.ReplyToMessageID = to.MessageID
	if err != nil {
		log.Println(err.Error())
	}
	return b.sendMsg(msg, buttons...)
}

func (b *Bot) replyErrorToUser(from *tgbotapi.User, text string, err error, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(from.ID, text+": "+err.Error())
	if err != nil {
		log.Println(err.Error())
	}
	return b.sendMsg(msg, buttons...)
}

func (b *Bot) sendToUser(userID int64, text string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(userID, text)
	return b.sendMsg(msg, buttons...)
}

func (b *Bot) sendTyping(to *tgbotapi.Message) {
	action := tgbotapi.NewChatAction(to.Chat.ID, tgbotapi.ChatTyping)
	_, err := b.bot.Send(action)
	if err != nil {
		log.Println("error while sending typing action: ", err)
	}
}

func (b *Bot) sendMsg(msg tgbotapi.MessageConfig, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	if len(buttons) > 0 {
		msg.ReplyMarkup = buildReplyMarkup(buttons...)
	}
	msg.ParseMode = tgbotapi.ModeHTML
	return b.send(msg)
}

func (b *Bot) sendPlainTextMsg(msg tgbotapi.MessageConfig, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	if len(buttons) > 0 {
		msg.ReplyMarkup = buildReplyMarkup(buttons...)
	}
	return b.send(msg)
}

func buildReplyMarkup(buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
	rowButtons := make([][]tgbotapi.InlineKeyboardButton, 0, len(buttons))
	for _, btn := range buttons {
		rowButtons = append(rowButtons, tgbotapi.NewInlineKeyboardRow(btn))
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		rowButtons...,
	)
}

func (b *Bot) send(msg tgbotapi.Chattable) tgbotapi.Message {
	replyMsg, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
	return replyMsg
}

func (b *Bot) reportError(errText string) {
	const reporter = 373512635
	go func() {
		b.sendMsg(tgbotapi.NewMessage(reporter, errText))
	}()
}

const (
	langCodeEn = "en"
	langCodeRu = "ru"
)

func getLanguageCode(user *tgbotapi.User) string {
	lang := langCodeEn
	if user.LanguageCode == langCodeRu {
		lang = langCodeRu
	}
	return lang
}

func DustToString(dust *service.Dust, spt string) string {
	var result string
	if dust.BlackCount > 0 {
		result += "🖤 " + strconv.FormatInt(dust.BlackCount, 10) + spt
	}
	if dust.WhiteCount > 0 {
		result += "🤍 " + strconv.FormatInt(dust.WhiteCount, 10) + spt
	}
	if dust.PurpleCount > 0 {
		result += "💜 " + strconv.FormatInt(dust.PurpleCount, 10) + spt
	}
	if dust.YellowCount > 0 {
		result += "💛 " + strconv.FormatInt(dust.YellowCount, 10) + spt
	}
	if dust.RedCount > 0 {
		result += "❤️ " + strconv.FormatInt(dust.RedCount, 10) + spt
	}
	if dust.OrangeCount > 0 {
		result += "🧡 " + strconv.FormatInt(dust.OrangeCount, 10) + spt
	}
	if dust.GreenCount > 0 {
		result += "💚 " + strconv.FormatInt(dust.GreenCount, 10) + spt
	}
	if dust.BlueCount > 0 {
		result += "💠 " + strconv.FormatInt(dust.BlueCount, 10) + spt
	}
	if dust.IndigoCount > 0 {
		result += "💙 " + strconv.FormatInt(dust.IndigoCount, 10) + spt
	}
	return result
}

func HerbToString(herb *service.Herb, spt string) string {
	var result string
	if herb.MelissaCount > 0 {
		result += "🌿 " + strconv.FormatInt(herb.MelissaCount, 10) + spt
	}
	if herb.LavandaCount > 0 {
		result += "🌱 " + strconv.FormatInt(herb.LavandaCount, 10) + spt
	}
	//☘️🌱🌿☘️🍀🍃🍂🍁🌾🎋🪴🎍
	return result
}

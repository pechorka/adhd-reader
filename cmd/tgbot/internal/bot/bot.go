package bot

import (
	"fmt"

	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/pechorka/adhd-reader/internal/service"
	"github.com/pechorka/adhd-reader/internal/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pechorka/adhd-reader/pkg/contenttype"
	"github.com/pechorka/adhd-reader/pkg/fileloader"
	"github.com/pechorka/adhd-reader/pkg/queue"
	"github.com/pechorka/adhd-reader/pkg/runeslice"
	"github.com/pechorka/adhd-reader/pkg/sizeconverter"
)

const (
	textSelect = "text-select:"
	deleteText = "delete-text:"
	nextChunk  = "next-chunk"
	prevChunk  = "prev-chunk"
)

const (
	defaultMaxFileSize = 20 * 1024 * 1024 // 20 MB
)

type Bot struct {
	service     *service.Service
	bot         *tgbotapi.BotAPI
	msgQueue    *queue.MessageQueue
	fileLoader  *fileloader.Loader
	maxFileSize int
}

type Config struct {
	Token       string
	Service     *service.Service
	MsgQueue    *queue.MessageQueue
	FileLoader  *fileloader.Loader
	MaxFileSize int
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

	return &Bot{
		service:     cfg.Service,
		bot:         bot,
		msgQueue:    cfg.MsgQueue,
		fileLoader:  cfg.FileLoader,
		maxFileSize: cfg.MaxFileSize,
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

func (b *Bot) handlePanic(msg *tgbotapi.Message) {
	if rec := recover(); rec != nil {
		b.replyWithText(msg, "Something went wrong, please try again later")
		b.sendMsg(tgbotapi.NewMessage(373512635, fmt.Sprintf("Я запаниковал: %v", rec)))
		log.Println("Panic: ", rec, "Stack: ", string(debug.Stack()))
	}
}

func (b *Bot) handleMsg(msg *tgbotapi.Message) {
	defer b.handlePanic(msg)

	if msg.Document != nil {
		b.saveTextFromDocument(msg)
		return
	}

	switch cmd := msg.Command(); cmd {
	case "start":
		b.start(msg)
	case "list":
		b.list(msg)
	case "page":
		b.page(msg)
	case "chunk":
		b.chunk(msg)
	case "delete":
		b.delete(msg)
	case "help":
		b.help(msg)
	default:
		if cmd != "" {
			log.Println("Unknown command: ", cmd)
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
		help - troubleshooting and support
	*/
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	defer b.handlePanic(cb.Message)

	switch {
	case strings.HasPrefix(cb.Data, textSelect):
		b.selectText(cb)
	case strings.HasPrefix(cb.Data, deleteText):
		b.deleteTextCallBack(cb)
	case cb.Data == nextChunk:
		b.nextChunk(cb)
	case cb.Data == prevChunk:
		b.prevChunk(cb)
	}
	// Respond to the callback query, telling Telegram to show the user
	// a message with the data received.
	callback := tgbotapi.NewCallback(cb.ID, cb.Data)
	if _, err := b.bot.Request(callback); err != nil {
		log.Println("Failed to respond to callback query:", err)
	}
}

func (b *Bot) selectText(cb *tgbotapi.CallbackQuery) {
	textUUID := strings.TrimPrefix(cb.Data, textSelect)
	currentText, err := b.service.SelectText(cb.From.ID, textUUID)
	if err != nil {
		b.replyError(cb.Message, "Failed to select text", err)
		return
	}
	msg := fmt.Sprintf("Current selected text is: <code>%s</code>", currentText.Name)
	b.replyWithText(cb.Message, msg)
	b.currentChunk(cb)
}

func (b *Bot) deleteTextCallBack(cb *tgbotapi.CallbackQuery) {
	textUUID := strings.TrimPrefix(cb.Data, deleteText)
	err := b.service.DeleteTextByUUID(cb.From.ID, textUUID)
	if err != nil {
		b.replyError(cb.Message, "Failed to delete text", err)
		return
	}
	b.replyWithText(cb.Message, textDeletedMsg)
}

func (b *Bot) nextChunk(cb *tgbotapi.CallbackQuery) {
	b.chunkReply(cb, b.service.NextChunk)
}

func (b *Bot) prevChunk(cb *tgbotapi.CallbackQuery) {
	b.chunkReply(cb, b.service.PrevChunk)
}

func (b *Bot) currentChunk(cb *tgbotapi.CallbackQuery) {
	b.chunkReply(cb, b.service.CurrentOrFirstChunk)
}

type chunkSelectorFunc func(userID int64) (storage.Text, string, service.ChunkType, error)

func (b *Bot) chunkReply(cb *tgbotapi.CallbackQuery, chunkSelector chunkSelectorFunc) {
	currentText, chunkText, chunkType, err := chunkSelector(cb.From.ID)
	prevBtn := tgbotapi.NewInlineKeyboardButtonData("Prev", prevChunk)
	nextBtn := tgbotapi.NewInlineKeyboardButtonData("Next", nextChunk)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("Delete text", deleteText+currentText.UUID)
	switch err {
	case service.ErrFirstChunk:
		b.replyWithText(cb.Message, "Can't go back, you are at the first chunk", nextBtn)
		return
	case service.ErrTextFinished:
		b.replyWithText(cb.Message, fmt.Sprintf("Text <code>%s</code> is finished", currentText.Name), prevBtn, deleteBtn)
	case nil:
	default:
		b.replyError(cb.Message, "Failed to get next chunk", err)
		return
	}

	switch chunkType {
	case service.ChunkTypeFirst:
		b.replyWithText(cb.Message, chunkText, nextBtn)
	case service.ChunkTypeLast:
		replyMsg := b.replyWithText(cb.Message, chunkText)
		b.replyWithText(&replyMsg, fmt.Sprintf("This was the last chunk from the text <code>%s</code>", currentText.Name), prevBtn, deleteBtn)
	default:
		b.replyWithText(cb.Message, chunkText, prevBtn, nextBtn)
	}
}

func (b *Bot) start(msg *tgbotapi.Message) {
	go func() { // todo: stop flow on other commands???
		b.sendToID(msg.From.ID, firstMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.sendToID(msg.From.ID, secondMsg)
		b.sendTyping(msg)
		time.Sleep(5 * time.Second)

		b.sendToID(msg.From.ID, thirdMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.sendToID(msg.From.ID, fourthMsg)
		b.sendToID(msg.From.ID, fifthMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.sendToID(msg.From.ID, sixthMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		fileMsg := tgbotapi.NewDocument(msg.From.ID, tgbotapi.FileBytes{
			Name:  startFileName,
			Bytes: startFile,
		})
		b.send(fileMsg)
		b.sendTyping(msg)
		time.Sleep(2 * time.Second)

		b.sendToID(msg.From.ID, eighthMsg)
	}()
}

func (b *Bot) list(msg *tgbotapi.Message) {
	texts, err := b.service.ListTexts(msg.From.ID)
	if err != nil {
		b.replyError(msg, "Failed to list texts", err)
		return
	}
	if len(texts) == 0 {
		b.replyWithText(msg, "No texts")
		return
	}
	// reply with button for each text and save text index in callback data
	var buttons []tgbotapi.InlineKeyboardButton
	for _, t := range texts {
		btnText := completionPercentString(t.CompletionPercent) + " " + t.Name
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(btnText, textSelect+t.UUID))
	}
	b.replyWithText(msg, "Select text to read", buttons...)
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

func (b *Bot) page(msg *tgbotapi.Message) {
	strPage := msg.CommandArguments()
	page, err := strconv.ParseInt(strings.TrimSpace(strPage), 10, 64)
	if err != nil {
		b.replyError(msg, "Failed to parse page", err)
		return
	}
	err = b.service.SetPage(msg.From.ID, page)
	if err != nil {
		if err == service.ErrTextNotSelected {
			b.replyWithText(msg, "Text not selected")
			return
		}
		b.replyError(msg, "Failed to set page", err)
		return
	}
	b.replyWithText(msg, "Page set")
}

func (b *Bot) chunk(msg *tgbotapi.Message) {
	strChunk := msg.CommandArguments()
	chunk, err := strconv.ParseInt(strings.TrimSpace(strChunk), 10, 64)
	if err != nil {
		b.replyError(msg, "Failed to parse chunk", err)
		return
	}
	err = b.service.SetChunkSize(msg.From.ID, chunk)
	if err != nil {
		b.replyError(msg, "Failed to set chunk size", err)
		return
	}
	b.replyWithText(msg, "Chunk size set. But keep in mind that text gets chunked on save and currently they are not re-chunked on chunk size change")
}

func (b *Bot) delete(msg *tgbotapi.Message) {
	textName := strings.TrimSpace(msg.CommandArguments())
	err := b.service.DeleteTextByName(msg.From.ID, textName)
	if err != nil {
		b.replyError(msg, "Failed to delete text", err)
		return
	}
	b.replyWithText(msg, textDeletedMsg)
}

func (b *Bot) help(msg *tgbotapi.Message) {
	b.replyWithText(msg, helpMsg)
}

func (b *Bot) saveTextFromDocument(msg *tgbotapi.Message) {
	if msg.Document.FileSize != 0 && msg.Document.FileSize > b.maxFileSize {
		b.replyWithText(msg, "File size is too big. Max file size is "+sizeconverter.HumanReadableSizeInMB(b.maxFileSize))
		return
	}
	if !contenttype.IsPlainText(msg.Document.MimeType) {
		b.replyWithText(msg, "Unsupported file format. Please input plain text file or send a message. We currently do not support other file formats.")
		return
	}
	fileURL, err := b.bot.GetFileDirectURL(msg.Document.FileID)
	if err != nil {
		b.replyError(msg, "Failed to build file url", err)
		return
	}
	text, err := b.fileLoader.DownloadTextFile(fileURL)
	switch err {
	case nil:
	case fileloader.ErrFileIsTooBig:
		b.replyWithText(msg, "File size is too big. Max file size is "+sizeconverter.HumanReadableSizeInMB(b.maxFileSize))
		return
	case fileloader.ErrNotPlainText:
		b.replyWithText(msg, "Unsupported file format. Please input plain text file or send a message. We currently do not support other file formats.")
		return
	default:
		b.replyError(msg, "Failed to download your file", err)
		return
	}

	textID, err := b.service.AddText(msg.From.ID, msg.Document.FileName, text)
	if err != nil {
		if err == service.ErrTextNotUTF8 {
			b.replyWithText(msg, "Text is not in UTF-8 encoding")
			return
		}
		b.replyError(msg, "Faled to save text", err)
		return
	}
	readBtn := tgbotapi.NewInlineKeyboardButtonData("Read", textSelect+textID)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("Delete", deleteText+textID)
	b.replyWithText(msg, fmt.Sprintf("Text <code>%s</code> is saved", msg.Document.FileName), readBtn, deleteBtn)
}

func (b *Bot) saveTextFromMessage(msg *tgbotapi.Message) {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	b.msgQueue.Add(msg.From.ID, text)
}

func (b *Bot) onQueueFilled(userID int64, msgText string) {
	textName := runeslice.NRunes(msgText, 64)
	textID, err := b.service.AddText(userID, textName, msgText)
	if err != nil {
		b.sendToID(userID, "Failed to save text: "+err.Error())
		return
	}
	readBtn := tgbotapi.NewInlineKeyboardButtonData("Read", textSelect+textID)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("Delete", deleteText+textID)
	b.sendToID(userID, fmt.Sprintf("Text <code>%s</code> is saved", textName), readBtn, deleteBtn)
}

func (b *Bot) replyWithText(to *tgbotapi.Message, text string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	return b.sendMsg(msg, buttons...)
}

func (b *Bot) replyError(to *tgbotapi.Message, text string, err error, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
	msg := tgbotapi.NewMessage(to.Chat.ID, text+": "+err.Error())
	msg.ReplyToMessageID = to.MessageID
	if err != nil {
		log.Println(err.Error())
	}
	return b.sendMsg(msg, buttons...)
}

func (b *Bot) sendToID(userID int64, text string, buttons ...tgbotapi.InlineKeyboardButton) tgbotapi.Message {
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
		rowButtons := make([][]tgbotapi.InlineKeyboardButton, 0, len(buttons))
		for _, btn := range buttons {
			rowButtons = append(rowButtons, tgbotapi.NewInlineKeyboardRow(btn))
		}
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			rowButtons...,
		)
	}
	msg.ParseMode = tgbotapi.ModeHTML
	return b.send(msg)
}

func (b *Bot) send(msg tgbotapi.Chattable) tgbotapi.Message {
	replyMsg, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
	return replyMsg
}
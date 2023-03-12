package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aakrasnova/zone-mate/pkg/fileloader"
	"github.com/aakrasnova/zone-mate/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	textSelect = "text-select:"
	deleteText = "delete-text:"
	nextChunk  = "next-chunk"
	prevChunk  = "prev-chunk"
)

type Bot struct {
	s   *service.Service
	bot *tgbotapi.BotAPI
}

func NewBot(s *service.Service, token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	bot.Debug = true // TODO before release take from config

	return &Bot{s: s, bot: bot}, nil
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

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
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
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
	err := b.s.SelectText(cb.From.ID, textUUID)
	if err != nil {
		b.replyError(cb.Message, "Failed to select text", err)
		return
	}
	msg := "Text selected successfully"
	currentText, err := b.s.CurrentText(cb.From.ID)
	if err == nil {
		msg = fmt.Sprintf("Current selected text is: <code>%s</code>", currentText.Name)
	}
	b.replyWithText(cb.Message, msg)
	b.nextChunk(cb)
}

func (b *Bot) deleteTextCallBack(cb *tgbotapi.CallbackQuery) {
	textUUID := strings.TrimPrefix(cb.Data, deleteText)
	err := b.s.DeleteTextByUUID(cb.From.ID, textUUID)
	if err != nil {
		b.replyError(cb.Message, "Failed to delete text", err)
		return
	}
	b.replyWithText(cb.Message, "Text deleted")
}

func (b *Bot) nextChunk(cb *tgbotapi.CallbackQuery) {
	prev := tgbotapi.NewInlineKeyboardButtonData("Prev", prevChunk)
	next := tgbotapi.NewInlineKeyboardButtonData("Next", nextChunk)
	text, err := b.s.NextChunk(cb.From.ID)
	if err != nil {
		if err == service.ErrTextFinished {
			buttons := []tgbotapi.InlineKeyboardButton{prev}
			currentText, err := b.s.CurrentText(cb.From.ID)
			if err == nil {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Delete text", deleteText+currentText.UUID))
			}
			b.replyWithText(cb.Message, fmt.Sprintf("Text <code>%s</code> is finished", currentText.Name), buttons...)
			return
		}
		b.replyError(cb.Message, "Failed to get next chunk", err)
		return
	}
	// reply chunk text with next/prev buttons
	b.replyWithText(cb.Message, text, prev, next)
}

func (b *Bot) prevChunk(cb *tgbotapi.CallbackQuery) {
	prev := tgbotapi.NewInlineKeyboardButtonData("Prev", prevChunk)
	next := tgbotapi.NewInlineKeyboardButtonData("Next", nextChunk)
	text, err := b.s.PrevChunk(cb.From.ID)
	if err != nil {
		if err == service.ErrFirstChunk {
			b.replyWithText(cb.Message, "Can't go back, you are at the first chunk", next)
			return
		}
		b.replyError(cb.Message, "Failed to get prev chunk", err)
		return
	}
	// reply chunk text with next/prev buttons
	b.replyWithText(cb.Message, text, prev, next)
}

func (b *Bot) handleMsg(msg *tgbotapi.Message) {
	defer func() {
		if rec := recover(); rec != nil {
			b.send(tgbotapi.NewMessage(373512635, fmt.Sprintf("Я запаниковал: %v", rec)))
		}
	}()

	if msg.Document != nil {
		b.saveTextFromDocument(msg)
		return
	}

	switch msg.Command() {
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
	default:
		b.saveTextFromMessage(msg)
	}

	// command for bot father to add command help
	/*
		/setcommands
		list - list all texts
		page - set page number, pass page number as argument
		chunk - set chunk size, pass chunk size as argument
		delete - delete text, pass text name as argument
	*/
}

func (b *Bot) start(msg *tgbotapi.Message) {
	b.replyWithText(msg, "I am working")
}

func (b *Bot) list(msg *tgbotapi.Message) {
	texts, err := b.s.ListTexts(msg.From.ID)
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
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(t.Name, textSelect+t.UUID))
	}
	b.replyWithText(msg, "Select text to read", buttons...)
}

func (b *Bot) page(msg *tgbotapi.Message) {
	strPage := msg.CommandArguments()
	page, err := strconv.ParseInt(strings.TrimSpace(strPage), 10, 64)
	if err != nil {
		b.replyError(msg, "Failed to parse page", err)
		return
	}
	err = b.s.SetPage(msg.From.ID, page)
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
	err = b.s.SetChunkSize(msg.From.ID, chunk)
	if err != nil {
		b.replyError(msg, "Failed to set chunk size", err)
		return
	}
	b.replyWithText(msg, "Chunk size set. But keep in mind that text gets chunked on save and currently they are not re-chunked on chunk size change")
}

func (b *Bot) delete(msg *tgbotapi.Message) {
	textName := strings.TrimSpace(msg.CommandArguments())
	err := b.s.DeleteTextByName(msg.From.ID, textName)
	if err != nil {
		b.replyError(msg, "Failed to delete text", err)
		return
	}
	b.replyWithText(msg, "Text deleted")
}

func (b *Bot) saveTextFromDocument(msg *tgbotapi.Message) {
	fileURL, err := b.bot.GetFileDirectURL(msg.Document.FileID)
	if err != nil {
		b.replyError(msg, "Failed to build file url", err)
		return
	}
	text, err := fileloader.DownloadTextFile(fileURL)
	if err != nil {
		b.replyError(msg, "Failed to download text file", err)
		return
	}
	textID, err := b.s.AddText(msg.From.ID, msg.Document.FileName, text)
	if err != nil {
		b.replyError(msg, "Faled to save text", err)
		return
	}
	readBtn := tgbotapi.NewInlineKeyboardButtonData("Read", textSelect+textID)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("Delete", deleteText+textID)
	b.replyWithText(msg, "This text is saved", readBtn, deleteBtn)
}

func (b *Bot) saveTextFromMessage(msg *tgbotapi.Message) {
	// first line is text name
	textName, _, ok := strings.Cut(msg.Text, "\n")
	if !ok {
		b.replyWithText(msg, "Text name not found (first line should be text name)")
		return
	}
	textID, err := b.s.AddText(msg.From.ID, textName, msg.Text)
	if err != nil {
		b.replyError(msg, "Faled to save text", err)
		return
	}
	readBtn := tgbotapi.NewInlineKeyboardButtonData("Read", textSelect+textID)
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("Delete", deleteText+textID)
	b.replyWithText(msg, "This text is saved", readBtn, deleteBtn)
}

func (b *Bot) replyWithText(to *tgbotapi.Message, text string, buttons ...tgbotapi.InlineKeyboardButton) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ParseMode = tgbotapi.ModeHTML
	b.send(msg, buttons...)
}

func (b *Bot) replyError(to *tgbotapi.Message, text string, err error, buttons ...tgbotapi.InlineKeyboardButton) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ParseMode = tgbotapi.ModeHTML
	if err != nil {
		log.Println(err.Error())
	}
	b.send(msg, buttons...)
}

func (b *Bot) send(msg tgbotapi.MessageConfig, buttons ...tgbotapi.InlineKeyboardButton) {
	if len(buttons) > 0 {
		rowButtons := make([][]tgbotapi.InlineKeyboardButton, 0, len(buttons))
		for _, btn := range buttons {
			rowButtons = append(rowButtons, tgbotapi.NewInlineKeyboardRow(btn))
		}
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			rowButtons...,
		)
	}
	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
}

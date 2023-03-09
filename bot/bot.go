package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aakrasnova/zone-mate/loader"
	"github.com/aakrasnova/zone-mate/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	textSelect = "text-select:"
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
	textIndex := strings.TrimPrefix(cb.Data, textSelect)
	textIndexInt, err := strconv.Atoi(textIndex)
	if err != nil {
		b.replyError(cb.Message, "Failed to select text", err)
		return
	}
	err = b.s.SelectText(cb.From.ID, textIndexInt)
	if err != nil {
		b.replyError(cb.Message, "Failed to select text", err)
		return
	}

	// todo: helper for sending markup
	markup := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("Start reading", nextChunk),
		},
	)
	replyMsg := tgbotapi.NewMessage(cb.From.ID, "Current text selected")
	replyMsg.ReplyMarkup = markup
	b.send(replyMsg)
}

func (b *Bot) nextChunk(cb *tgbotapi.CallbackQuery) {
	text, err := b.s.NextChunk(cb.From.ID)
	if err != nil {
		if err == service.ErrTextFinished {
			markup := tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData("Prev", prevChunk),
				},
			)
			replyMsg := tgbotapi.NewMessage(cb.From.ID, "Text finished")
			replyMsg.ReplyMarkup = markup
			b.send(replyMsg)
			return
		}
		b.replyError(cb.Message, "Failed to get next chunk", err)
		return
	}
	// reply chunk text with next/prev buttons
	markup := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("Prev", prevChunk),
			tgbotapi.NewInlineKeyboardButtonData("Next", nextChunk),
		},
	)
	replyMsg := tgbotapi.NewMessage(cb.From.ID, text)
	replyMsg.ReplyMarkup = markup
	b.send(replyMsg)
}

func (b *Bot) prevChunk(cb *tgbotapi.CallbackQuery) {
	text, err := b.s.PrevChunk(cb.From.ID)
	if err != nil {
		if err == service.ErrFirstChunk {
			markup := tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData("Next", nextChunk),
				},
			)
			replyMsg := tgbotapi.NewMessage(cb.From.ID, "Can't go back, you are at the first chunk")
			replyMsg.ReplyMarkup = markup
			b.send(replyMsg)
			return
		}
		b.replyError(cb.Message, "Failed to get prev chunk", err)
		return
	}
	// reply chunk text with next/prev buttons
	markup := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("Prev", prevChunk),
			tgbotapi.NewInlineKeyboardButtonData("Next", nextChunk),
		},
	)
	replyMsg := tgbotapi.NewMessage(cb.From.ID, text)
	replyMsg.ReplyMarkup = markup
	b.send(replyMsg)
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
	}
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
	for i, t := range texts {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(t, fmt.Sprintf("%s%d", textSelect, i)))
	}
	// todo: helper for sending markup
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)
	replyMsg := tgbotapi.NewMessage(msg.Chat.ID, "Select text")
	replyMsg.ReplyMarkup = markup
	replyMsg.ReplyToMessageID = msg.MessageID
	b.send(replyMsg)
}

func (b *Bot) page(msg *tgbotapi.Message) {
	strPage := msg.CommandArguments()
	page, err := strconv.Atoi(strings.TrimSpace(strPage))
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

func (b *Bot) saveTextFromDocument(msg *tgbotapi.Message) {
	fileURL, err := b.bot.GetFileDirectURL(msg.Document.FileID)
	if err != nil {
		b.replyError(msg, "Failed to build file url", err)
		return
	}
	text, err := loader.DownloadTextFile(fileURL)
	if err != nil {
		b.replyError(msg, "Failed to download text file", err)
		return
	}
	err = b.s.AddText(msg.From.ID, msg.Document.FileName, text)
	if err != nil {
		b.replyError(msg, "Faled to save text", err)
		return
	}
	b.replyWithText(msg, "Successfully saved text")
}

func (b *Bot) replyWithText(to *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	msg.ParseMode = tgbotapi.ModeHTML
	b.send(msg)
}

func (b *Bot) replyError(to *tgbotapi.Message, text string, err error) {
	msg := tgbotapi.NewMessage(to.Chat.ID, text)
	msg.ReplyToMessageID = to.MessageID
	if err != nil {
		log.Println(err.Error())
	}
	b.send(msg)
}

func (b *Bot) send(msg tgbotapi.MessageConfig) {
	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println("error while sending message: ", err)
	}
}

package bot

import (
	"fmt"
	"log"

	"github.com/aakrasnova/zone-mate/loader"
	"github.com/aakrasnova/zone-mate/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	}
}

func (b *Bot) Stop() {
	b.bot.StopReceivingUpdates()
}

func (b *Bot) handleMsg(msg *tgbotapi.Message) {
	// defer func() {
	// 	if rec := recover(); rec != nil {
	// 		b.send(tgbotapi.NewMessage(373512635, fmt.Sprintf("Я запаниковал: %v", rec)))
	// 	}
	// }()

	if msg.Document != nil {
		b.saveTextFromDocument(msg)
		return
	}

	switch msg.Command() {
	case "start":
		b.start(msg)
	case "list":
		b.list(msg)
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
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(t, fmt.Sprintf("%d", i)))
	}
	// todo: helper for sending markup
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)
	replyMsg := tgbotapi.NewMessage(msg.Chat.ID, "Select text")
	replyMsg.ReplyMarkup = markup
	replyMsg.ReplyToMessageID = msg.MessageID
	b.send(replyMsg)
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

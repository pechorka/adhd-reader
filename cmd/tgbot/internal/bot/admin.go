package bot

import (
	"bytes"
	"encoding/csv"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) analytics(msg *tgbotapi.Message) {
	analytics, err := b.service.Analytics()
	if err != nil {
		b.replyError(msg, "could not fetch analytics", err)
		return
	}
	// create csv file with analytics
	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)

	header := []string{"UserID", "ChunkSize", "TotalTextCount", "AvgTotalChunks", "MaxCurrentChunk", "StartedTextsCount", "CompletedTextsCount", "CurrentTextName"}
	if err := writer.Write(header); err != nil {
		b.replyError(msg, "could not write csv header", err)
		return
	}
	for _, ua := range analytics {
		row := []string{
			fmt.Sprintf("%d", ua.UserID),
			fmt.Sprintf("%d", ua.ChunkSize),
			fmt.Sprintf("%d", ua.TotalTextCount),
			fmt.Sprintf("%d", ua.AvgTotalChunks),
			fmt.Sprintf("%d", ua.MaxCurrentChunk),
			fmt.Sprintf("%d", ua.StartedTextsCount),
			fmt.Sprintf("%d", ua.CompletedTextsCount),
			ua.CurrentTextName,
		}

		if err := writer.Write(row); err != nil {
			b.replyError(msg, "could not write csv row", err)
			return
		}
	}
	writer.Flush()

	doc := tgbotapi.FileBytes{Name: "user_analytics.csv", Bytes: buffer.Bytes()}
	b.send(tgbotapi.NewDocument(msg.Chat.ID, doc))
}

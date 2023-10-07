package fb2

import (
	"bytes"

	"github.com/PuerkitoBio/goquery"
)

func PlainText(data []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	body := doc.Find("body")

	return body.Text(), nil
}

package pdfexctractor

import (
	"bytes"

	"github.com/ledongthuc/pdf"
)

func ExtractPlainTextFromPDF(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	totalPage := r.NumPage()

	var buf bytes.Buffer
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		rows, _ := p.GetTextByRow()
		for _, row := range rows {
			for _, word := range row.Content {
				buf.WriteString(word.S)
			}
		}
	}

	return buf.String(), nil
}

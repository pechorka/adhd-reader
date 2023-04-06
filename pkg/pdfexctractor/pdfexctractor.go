package pdfexctractor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"

	stdErrs "errors"

	"github.com/ledongthuc/pdf"
	"github.com/pkg/errors"
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

func ExtractPlainTextFromPDF_PdfToText(data []byte) (_ string, err error) {
	fName := fmt.Sprintf("/tmp/temp-%d.pdf", time.Now().UnixNano())
	err = os.WriteFile(fName, data, 0644)
	if err != nil {
		return "", errors.Wrap(err, "failed to write temp file")
	}
	defer func() { err = stdErrs.Join(os.Remove(fName), err) }()

	cmd := exec.Command("pdftotext", fName, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to extract text from pdf")
	}

	return string(out), nil
}

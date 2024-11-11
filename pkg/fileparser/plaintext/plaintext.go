package plaintext

import (
	"unicode/utf8"

	"github.com/pechorka/gostdlib/pkg/errs"
)

func PlainText(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", errs.New("text is not valid utf8")
	}
	return string(data), nil
}

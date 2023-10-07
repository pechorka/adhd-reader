package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	stdErrs "errors"

	"github.com/pkg/errors"
)

func PlaintText(data []byte) (_ string, err error) {
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

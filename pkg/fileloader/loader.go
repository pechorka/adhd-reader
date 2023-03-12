package fileloader

import (
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// LoadTextFile downloads a text file from the given URL and returns its content.
func DownloadTextFile(URL string) (string, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return "", errors.Wrap(err, "failed to download text file")
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read text file")
	}

	return string(content), nil
}

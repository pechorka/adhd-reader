package fileloader

import (
	"io"
	"net/http"
	"time"

	"github.com/pechorka/adhd-reader/pkg/sizeconverter"
	"github.com/pkg/errors"
)

const (
	defaultMaxFileSize   = 20 * 1024 * 1024 // 20 MB
	defaultClientTimeout = 10 * time.Second
)

type Loader struct {
	maxFileSize int64 // in bytes
	maxSizeErr  error

	httpCli *http.Client
}

type Config struct {
	MaxFileSize int64
	HttpTimeout time.Duration
}

func NewLoader(cfg Config) *Loader {
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = defaultMaxFileSize
	}
	if cfg.HttpTimeout == 0 {
		cfg.HttpTimeout = defaultClientTimeout
	}
	return &Loader{
		maxFileSize: cfg.MaxFileSize,
		maxSizeErr:  errors.New("file is too big, max size is " + sizeconverter.HumanReadableSizeInMB(cfg.MaxFileSize)),
		httpCli: &http.Client{
			Timeout: cfg.HttpTimeout,
		},
	}
}

// DownloadTextFile downloads a file from the given URL and returns its content.
func (l *Loader) DownloadTextFile(URL string) (string, error) {
	resp, err := l.httpCli.Get(URL)
	if err != nil {
		return "", errors.Wrap(err, "failed to download file")
	}
	defer resp.Body.Close()
	if ctLen := resp.ContentLength; ctLen != -1 && ctLen > l.maxFileSize {
		return "", l.maxSizeErr
	}
	limitedReader := http.MaxBytesReader(nil, resp.Body, l.maxFileSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return "", l.maxSizeErr
		}
		return "", errors.Wrap(err, "failed to read file")
	}

	return string(content), nil
}

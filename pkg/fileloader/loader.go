package fileloader

import (
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultMaxFileSize   = 50 * 1024 * 1024 // 20 MB
	defaultClientTimeout = 10 * time.Second
)

var ErrFileIsTooBig = errors.New("file is too big")

type Loader struct {
	maxFileSize int64 // in bytes

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
		httpCli: &http.Client{
			Timeout: cfg.HttpTimeout,
		},
	}
}

func (l *Loader) DownloadFile(URL string) ([]byte, error) {
	resp, err := l.httpCli.Get(URL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download file")
	}
	defer resp.Body.Close()
	if ctLen := resp.ContentLength; ctLen != -1 && ctLen > l.maxFileSize {
		return nil, ErrFileIsTooBig
	}
	limitedReader := http.MaxBytesReader(nil, resp.Body, l.maxFileSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, ErrFileIsTooBig
		}
		return nil, errors.Wrap(err, "failed to read file")
	}

	return content, nil
}

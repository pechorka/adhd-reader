package fileloader

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestLoader_DownloadTextFile(t *testing.T) {
	fs := fstest.MapFS{
		"10MbFile.txt": {
			Data: bytes.Repeat([]byte("a"), 10*1024*1024),
		},
		"20MbFile.txt": {
			Data: bytes.Repeat([]byte("a"), 20*1024*1024),
		},
		"30MbFile.txt": {
			Data: bytes.Repeat([]byte("a"), 30*1024*1024),
		},
	}
	srv := httptest.NewServer(http.FileServer(http.FS(fs)))
	t.Cleanup(srv.Close)

	loader := NewLoader(Config{
		MaxFileSize: 20 * 1024 * 1024,
	})

	t.Run("10MbFile", func(t *testing.T) {
		_, err := loader.DownloadFile(srv.URL + "/10MbFile.txt")
		require.NoError(t, err)
	})

	t.Run("20MbFile", func(t *testing.T) {
		_, err := loader.DownloadFile(srv.URL + "/20MbFile.txt")
		require.NoError(t, err)
	})

	t.Run("30MbFile", func(t *testing.T) {
		_, err := loader.DownloadFile(srv.URL + "/30MbFile.txt")
		require.Error(t, err)
		require.True(t, strings.HasPrefix(err.Error(), "file is too big"))
	})
}

func TestLoader_DownloadTextFile_NoContentLentgh(t *testing.T) {
	// serve 30Mb file without Content-Length header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "")
		w.Write(bytes.Repeat([]byte("a"), 30*1024*1024))
	}))
	t.Cleanup(srv.Close)

	loader := NewLoader(Config{
		MaxFileSize: 20 * 1024 * 1024,
	})

	_, err := loader.DownloadFile(srv.URL)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "file is too big"))
}

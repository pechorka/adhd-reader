package chunks_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pechorka/adhd-reader/internal/storage/chunks"
	"github.com/stretchr/testify/require"
)

func TestAddText(t *testing.T) {
	t.Run("create -> get", func(t *testing.T) {
		db := testStorage(t)
		so := require.New(t)

		text := `1,2,3`
		chunks := strings.Split(text, ",")

		id, err := db.AddText(text, chunks)
		so.NoError(err)
		so.NotEmpty(id)

		chunk, err := db.GetChunk(id, 0)
		so.NoError(err)
		so.Equal("1", chunk)

		chunk, err = db.GetChunk(id, 1)
		so.NoError(err)
		so.Equal("2", chunk)

		chunk, err = db.GetChunk(id, 2)
		so.NoError(err)
		so.Equal("3", chunk)

		_, err = db.GetChunk(id, 3)
		so.Error(err)
	})

	t.Run("create -> delete", func(t *testing.T) {
		db := testStorage(t)
		so := require.New(t)

		text := `1,2,3`
		chunks := strings.Split(text, ",")

		id, err := db.AddText(text, chunks)
		so.NoError(err)
		so.NotEmpty(id)

		err = db.DeleteTextByUUID(id)
		so.NoError(err)

		_, err = db.GetChunk(id, 0)
		so.Error(err)
	})
}

func testStorage(t *testing.T) *chunks.Storage {
	t.Helper()

	path := filepath.Join(os.TempDir(), fmt.Sprintf("adhd-reader-%s.db", uuid.New().String()))

	storage, err := chunks.NewStorage(path)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := storage.Close()
		require.NoError(t, err)
	})

	return storage
}

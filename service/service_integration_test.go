package service

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/aakrasnova/zone-mate/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ListTexts(t *testing.T) {
	srv := NewService(testStorage(t), 100)
	userID := rand.Int63()

	err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	err = srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	err = srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	texts, err := srv.ListTexts(userID)
	require.NoError(t, err)
	require.Equal(t, []string{"text1Name", "text2Name", "text3Name"}, texts)
}

func TestService_SelectText(t *testing.T) {
	srv := NewService(testStorage(t), 100)
	userID := rand.Int63()

	err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	err = srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	err = srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	err = srv.SelectText(userID, -1)
	require.Error(t, err)
	err = srv.SelectText(userID, 3)
	require.Error(t, err)

	for i := 0; i < 3; i++ {
		err = srv.SelectText(userID, i)
		require.NoError(t, err)
		texts, err := srv.s.GetTexts(userID)
		require.NoError(t, err)
		require.Equal(t, i, texts.Current)
	}
}

func TestService_DeleteText(t *testing.T) {
	srv := NewService(testStorage(t), 100)
	userID := rand.Int63()

	err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	err = srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	err = srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	texts, err := srv.ListTexts(userID)
	require.NoError(t, err)
	require.Equal(t, []string{"text1Name", "text2Name", "text3Name"}, texts)

	err = srv.DeleteText(userID, "nonexistent")
	require.Error(t, err)

	err = srv.DeleteText(userID, "text2Name")
	require.NoError(t, err)

	texts, err = srv.ListTexts(userID)
	require.NoError(t, err)
	require.Equal(t, []string{"text1Name", "text3Name"}, texts)
}

func TestService_PageNavigation(t *testing.T) {
	srv := NewService(testStorage(t), 5)
	userID := rand.Int63()
	err := srv.AddText(
		userID, "textName",
		`First chunk.Second chunk.
		Third chunk.Fourth chunk.`,
	)
	require.NoError(t, err)
	err = srv.SelectText(userID, 0)
	require.NoError(t, err)

	chunks := []string{
		"First chunk.", "Second chunk.", "Third chunk.", "Fourth chunk.",
	}

	for i := 0; i < len(chunks); i++ {
		chunkText, err := srv.NextChunk(userID)
		require.NoError(t, err)
		require.Equal(t, chunks[i], chunkText)
	}
	_, err = srv.NextChunk(userID)
	require.Equal(t, ErrTextFinished, err)

	// -2, because we are currently on the last chunk.
	// So prev chunk should the 3rd one.
	for i := len(chunks) - 2; i >= 0; i-- {
		chunkText, err := srv.PrevChunk(userID)
		require.NoError(t, err)
		require.Equal(t, chunks[i], chunkText)
	}
	_, err = srv.PrevChunk(userID)
	require.Equal(t, ErrFirstChunk, err)
}

func TestService_SetPage(t *testing.T) {
	srv := NewService(testStorage(t), 5)
	userID := rand.Int63()
	err := srv.AddText(
		userID, "textName",
		`First chunk.Second chunk.
		Third chunk.Fourth chunk.`,
	)
	require.NoError(t, err)

	// no text selected
	err = srv.SetPage(userID, 0)
	require.Error(t, err)

	err = srv.SelectText(userID, 0)
	require.NoError(t, err)

	// page out of range
	err = srv.SetPage(userID, -1)
	require.Error(t, err)
	err = srv.SetPage(userID, 4)
	require.Error(t, err)

	// page in range
	for i := 0; i < 4; i++ {
		err = srv.SetPage(userID, int64(i))
		require.NoError(t, err)
	}
}

func testStorage(t *testing.T) *storage.Storage {
	t.Helper()
	storage, err := storage.NewStorage(fmt.Sprintf("/tmp/zone-mate-test-%d.db", rand.Int63()))
	require.NoError(t, err)
	t.Cleanup(func() {
		closeErr := storage.Close()
		assert.NoError(t, closeErr)
	})

	return storage
}

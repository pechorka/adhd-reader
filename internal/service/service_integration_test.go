package service

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/pechorka/adhd-reader/internal/storage"
	"github.com/pechorka/adhd-reader/pkg/chance"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ListTexts(t *testing.T) {
	srv := NewService(testStorage(t), 100, nil, nil)
	userID := rand.Int63()

	text1ID, err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	text2ID, err := srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	text3ID, err := srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	texts, more, err := srv.ListTexts(userID, 1, 50)
	require.NoError(t, err)
	require.False(t, more)
	require.Equal(t, text1ID, texts[0].UUID)
	require.Equal(t, text2ID, texts[1].UUID)
	require.Equal(t, text3ID, texts[2].UUID)
}

func TestService_ListTextsPagination(t *testing.T) {
	srv := NewService(testStorage(t), 100, nil, nil)
	userID := rand.Int63()

	text1ID, err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	text2ID, err := srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	text3ID, err := srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	texts, more, err := srv.ListTexts(userID, 1, 1)
	require.NoError(t, err)
	require.True(t, more)
	require.Len(t, texts, 1)
	require.Equal(t, text1ID, texts[0].UUID)

	texts, more, err = srv.ListTexts(userID, 2, 1)
	require.NoError(t, err)
	require.True(t, more)
	require.Len(t, texts, 1)
	require.Equal(t, text2ID, texts[0].UUID)

	texts, more, err = srv.ListTexts(userID, 3, 1)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, texts, 1)
	require.Equal(t, text3ID, texts[0].UUID)
}

func TestService_SelectText(t *testing.T) {
	srv := NewService(testStorage(t), 100, nil, nil)
	userID := rand.Int63()

	text1ID, err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	text2ID, err := srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	text3ID, err := srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	_, err = srv.SelectText(userID, "wrong uuid")
	require.Error(t, err)

	text1, err := srv.SelectText(userID, text1ID)
	require.NoError(t, err)
	require.Equal(t, "text1Name", text1.Name)

	text2, err := srv.SelectText(userID, text2ID)
	require.NoError(t, err)
	require.Equal(t, "text2Name", text2.Name)

	text3, err := srv.SelectText(userID, text3ID)
	require.NoError(t, err)
	require.Equal(t, "text3Name", text3.Name)
}

func TestService_DeleteTextByUUID(t *testing.T) {
	srv := NewService(testStorage(t), 100, nil, nil)
	userID := rand.Int63()

	text1ID, err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	text2ID, err := srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	text3ID, err := srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	texts, more, err := srv.ListTexts(userID, 1, 50)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, texts, 3)
	require.Equal(t, text1ID, texts[0].UUID)
	require.Equal(t, text2ID, texts[1].UUID)
	require.Equal(t, text3ID, texts[2].UUID)

	err = srv.DeleteTextByUUID(userID, "nonexistent")
	require.Error(t, err)

	err = srv.DeleteTextByUUID(userID, text2ID)
	require.NoError(t, err)

	texts, more, err = srv.ListTexts(userID, 1, 50)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, texts, 2)
	require.Equal(t, text1ID, texts[0].UUID)
	require.Equal(t, text3ID, texts[1].UUID)
}

func TestService_DeleteTextByName(t *testing.T) {
	srv := NewService(testStorage(t), 100, nil, nil)
	userID := rand.Int63()

	_, err := srv.AddText(userID, "text1Name", "text1")
	require.NoError(t, err)
	_, err = srv.AddText(userID, "text2Name", "text2")
	require.NoError(t, err)
	_, err = srv.AddText(userID, "text3Name", "text3")
	require.NoError(t, err)

	texts, more, err := srv.ListTexts(userID, 1, 50)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, texts, 3)
	require.Equal(t, "text1Name", texts[0].Name)
	require.Equal(t, "text2Name", texts[1].Name)
	require.Equal(t, "text3Name", texts[2].Name)

	err = srv.DeleteTextByName(userID, "nonexistent")
	require.Error(t, err)

	err = srv.DeleteTextByName(userID, "text2Name")
	require.NoError(t, err)

	texts, more, err = srv.ListTexts(userID, 1, 50)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, texts, 2)
	require.Equal(t, "text1Name", texts[0].Name)
	require.Equal(t, "text3Name", texts[1].Name)
}

func TestService_PageNavigation(t *testing.T) {
	srv := NewService(testStorage(t), 5, nil, nil)
	userID := rand.Int63()
	textID, err := srv.AddText(
		userID, "textName",
		`First chunk.Second chunk.
		Third chunk.Fourth chunk.`,
	)
	require.NoError(t, err)
	_, err = srv.SelectText(userID, textID)
	require.NoError(t, err)

	chunks := []string{
		"First chunk.", "Second chunk.", "Third chunk.", "Fourth chunk.",
	}

	for i := 0; i < len(chunks); i++ {
		_, chunkText, _, err := srv.NextChunk(userID)
		require.NoError(t, err)
		require.Equal(t, chunks[i], chunkText)
	}
	_, _, _, err = srv.NextChunk(userID)
	require.Equal(t, ErrTextFinished, err)

	// -2, because we are currently on the last chunk.
	// So prev chunk should the 3rd one.
	for i := len(chunks) - 2; i >= 0; i-- {
		_, chunkText, _, err := srv.PrevChunk(userID)
		require.NoError(t, err)
		require.Equal(t, chunks[i], chunkText)
	}
	_, _, _, err = srv.PrevChunk(userID)
	require.Equal(t, ErrFirstChunk, err)
}

func TestService_SetPage(t *testing.T) {
	srv := NewService(testStorage(t), 5, nil, nil)
	userID := rand.Int63()
	textID, err := srv.AddText(
		userID, "textName",
		`First chunk.Second chunk.
		Third chunk.Fourth chunk.`,
	)
	require.NoError(t, err)

	// no text selected
	err = srv.SetPage(userID, 0)
	require.Error(t, err)

	_, err = srv.SelectText(userID, textID)
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

func TestService_SetChunkSize(t *testing.T) {
	store := testStorage(t)
	srv := NewService(store, 5, nil, nil)
	userID := rand.Int63()

	err := srv.SetChunkSize(userID, -1)
	require.Error(t, err)
	err = srv.SetChunkSize(userID, telegramMessageLengthLimit+1)
	require.Error(t, err)

	err = srv.SetChunkSize(userID, 5)
	require.NoError(t, err)
}

func TestDustOnNextChunk(t *testing.T) {
	t.Run("dust is added", func(t *testing.T) {
		store := testStorage(t)
		srv := NewService(store, 5, nil, nil)
		srv.chancer = &mockChancer{
			winResult:          true,
			pickWinResultIndex: 0, // index of red dust
		}

		userID := rand.Int63()

		loot, err := srv.LootOnNextChunk(userID)
		require.NoError(t, err)
		require.EqualValues(t, 1, loot.TotalDust.RedCount)
		require.EqualValues(t, 1, loot.DeltaDust.RedCount)
	})

	t.Run("dust is not added", func(t *testing.T) {
		store := testStorage(t)
		srv := NewService(store, 5, nil, nil)
		srv.chancer = &mockChancer{
			winResult: false,
		}

		userID := rand.Int63()

		loot, err := srv.LootOnNextChunk(userID)
		require.NoError(t, err)
		require.EqualValues(t, 0, loot.TotalDust.RedCount)
	})
}

func testStorage(t *testing.T) *storage.Storage {
	t.Helper()
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("adhd-reader-test-%d.db", rand.Int63()))
	storage, err := storage.NewStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		closeErr := storage.Close()
		assert.NoError(t, closeErr)
		removeErr := os.Remove(dbPath)
		assert.NoError(t, removeErr)
	})

	return storage
}

type mockChancer struct {
	winResult          bool
	pickWinResultIndex int
}

func (m *mockChancer) Win(percent float64) bool {
	return m.winResult
}

func (m *mockChancer) PickWin(inputs ...chance.WinInput) {
	inputs[m.pickWinResultIndex].Action()
}

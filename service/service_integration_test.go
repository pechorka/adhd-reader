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

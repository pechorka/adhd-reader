package storage_test

// import (
// 	"testing"

// 	"github.com/google/uuid"
// 	"github.com/pechorka/adhd-reader/internal/storage"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestAddText(t *testing.T) {
// 	s, err := storage.NewTempStorage()
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		err := s.Close()
// 		require.NoError(t, err)
// 	})

// 	userID := int64(1)
// 	newText := storage.NewText{
// 		Name:   "Test text",
// 		Text:   "This is a test text.",
// 		Chunks: []string{"This is a test text."},
// 	}

// 	// Test adding a new text
// 	textUUID, err := s.AddText(userID, newText)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, textUUID)

// 	// Verify the added text
// 	textsWithChunkInfo, err := s.GetTexts(userID)
// 	require.NoError(t, err)
// 	require.Len(t, textsWithChunkInfo, 1)

// 	addedText := textsWithChunkInfo[0]
// 	assert.Equal(t, newText.Name, addedText.Name)
// 	assert.Equal(t, int64(1), addedText.TotalChunks)
// 	assert.Equal(t, int64(storage.NotSelected), addedText.CurrentChunk)
// 	assert.Equal(t, storage.SourceText, addedText.Source)

// 	// Test adding a text with the same name
// 	_, err = s.AddText(userID, newText)
// 	assert.Error(t, err)

// 	// Test adding a text with a different name
// 	newText2 := storage.NewText{
// 		Name:   "Another test text",
// 		Text:   "This is another test text.",
// 		Chunks: []string{"This is another test text."},
// 	}
// 	textUUID2, err := s.AddText(userID, newText2)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, textUUID2)
// 	assert.NotEqual(t, textUUID, textUUID2)

// 	// Verify that there are now two texts
// 	textsWithChunkInfo, err = s.GetTexts(userID)
// 	require.NoError(t, err)
// 	require.Len(t, textsWithChunkInfo, 2)
// }

// func TestAddTextFromProcessedFile(t *testing.T) {
// 	// Test setup
// 	s, err := storage.NewTempStorage()
// 	require.NoError(t, err)
// 	defer s.Close()

// 	userID := int64(1)
// 	name := "Test Text"
// 	pf, err := s.AddProcessedFile(storage.NewProcessedFile{
// 		Text:      "This is a test text.",
// 		Chunks:    []string{"This is a test text."},
// 		ChunkSize: 300,
// 		CheckSum:  []byte("1234567890"),
// 	})
// 	require.NoError(t, err)

// 	// successful add new text
// 	{
// 		addedUUID, err := s.AddTextFromProcessedFile(userID, name, pf)
// 		require.NoError(t, err)
// 		assert.Equal(t, pf.UUID, addedUUID)

// 		texts, err := s.GetTexts(userID)
// 		require.NoError(t, err)
// 		require.Len(t, texts, 1)
// 		assert.Equal(t, pf.UUID, texts[0].UUID)
// 		assert.Equal(t, name, texts[0].Name)
// 		assert.Equal(t, storage.SourceFile, texts[0].Source)
// 	}

// 	// fail to add text with the same name
// 	{
// 		_, err := s.AddTextFromProcessedFile(userID, name, pf)
// 		require.Error(t, err)
// 	}

// 	// fail to add text with the same UUID
// 	{
// 		_, err := s.AddTextFromProcessedFile(userID, "Another Test Text", pf)
// 		require.Error(t, err)
// 	}
// }

// func TestUpdateTexts(t *testing.T) {
// 	// Test setup
// 	s, err := storage.NewTempStorage()
// 	require.NoError(t, err)
// 	defer s.Close()

// 	userID := int64(1)
// 	newName := "Updated Text"

// 	addedUUID, err := s.AddText(userID, storage.NewText{
// 		Name:      "Original Text",
// 		Text:      "This is a test text.",
// 		Chunks:    []string{"This is a test text."},
// 		ChunkSize: 300,
// 	})
// 	require.NoError(t, err)

// 	// successful update
// 	{
// 		err := s.UpdateCurrentText(userID, func(t *storage.Text) error {
// 			t.Name = newName
// 			return nil
// 		})
// 		require.NoError(t, err)

// 		texts, err := s.GetTexts(userID)
// 		require.NoError(t, err)
// 		require.Len(t, texts, 1)
// 		assert.Equal(t, addedUUID, texts[0].UUID)
// 		assert.Equal(t, newName, texts[0].Name)
// 	}

// 	t.Run("nonexistent_uuid_error", func(t *testing.T) {
// 		nonExistentUUID := uuid.New().String()
// 		err := s.UpdateTexts(userID, nonExistentUUID, "New Name")
// 		require.Error(t, err)
// 	})

// 	t.Run("duplicate_name_error", func(t *testing.T) {
// 		anotherPF := storage.ProcessedFile{
// 			UUID:       uuid.New().String(),
// 			BucketName: []byte("another-test-bucket"),
// 			ChunkSize:  50,
// 			CheckSum:   []byte("0987654321"),
// 		}
// 		anotherUUID, err := s.AddTextFromProcessedFile(userID, "Another Text", anotherPF)
// 		require.NoError(t, err)

// 		err = s.UpdateTexts(userID, anotherUUID, newName)
// 		require.Error(t, err)
// 	})
// }

package storage

import (
	"encoding/binary"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

const NotSelected = -1

var (
	bktTexts = []byte("texts")
)

var (
	fullTextKey     = []byte("full_text")
	currentChunkKey = []byte("current_chunk")
	totalChunksKey  = []byte("total_chunks")
)

// Storage is a wrapper around bolt.DB
type Storage struct {
	db *bolt.DB
}

// NewStorage creates a new storage
func NewStorage(path string) (*Storage, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Storage{db: db}, nil
}

// Close closes the storage
func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) AddText(userID int64, newText NewText) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// fill the text bucket
		textBucketName := []byte(uuid.New().String())
		textBucket, err := tx.CreateBucketIfNotExists(textBucketName)
		if err != nil {
			return err
		}
		if err = textBucket.Put(fullTextKey, []byte(newText.Text)); err != nil {
			return err
		}
		if err = textBucket.Put(currentChunkKey, int64ToBytes(NotSelected)); err != nil {
			return err
		}
		if err = textBucket.Put(totalChunksKey, int64ToBytes(int64(len(newText.Chunks)))); err != nil {
			return err
		}
		for i, chunk := range newText.Chunks {
			if err = textBucket.Put(int64ToBytes(int64(i)), []byte(chunk)); err != nil {
				return err
			}
		}
		// update user bucket
		b, err := tx.CreateBucketIfNotExists(bktTexts)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		texts.Texts = append(texts.Texts, Text{
			Name:       newText.Name,
			BucketName: textBucketName,
		})
		if err = putTexts(b, id, texts); err != nil {
			return err
		}
		return nil
	})
}

func (s *Storage) GetTexts(id int64) (UserTexts, error) {
	var texts UserTexts
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktTexts)
		if b == nil {
			return nil
		}
		var err error
		texts, err = getTexts(b, int64ToBytes(id))
		return err
	})
	return texts, err
}

func (s *Storage) UpdateTexts(userID int64, updFunc func(*UserTexts) error) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktTexts)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		if err = updFunc(&texts); err != nil {
			return err
		}
		return putTexts(b, id, texts)
	})
}

func (s *Storage) SelectChunk(userID int64, updFunc func(curChunk, totalChunks int64) (nextChunk int64, err error)) (string, error) {
	var chunkText string
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktTexts)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		if texts.Current == NotSelected {
			return errors.New("no text selected")
		}
		textBucket := tx.Bucket(texts.Texts[texts.Current].BucketName)
		if textBucket == nil { // should not happen
			return errors.New("unexpected error: text bucket not found")
		}
		curChunk := bytesToInt64(textBucket.Get(currentChunkKey))
		totalChunks := bytesToInt64(textBucket.Get(totalChunksKey))
		nextChunk, err := updFunc(curChunk, totalChunks)
		if err != nil {
			return err
		}
		err = textBucket.Put(currentChunkKey, int64ToBytes(nextChunk))
		if err != nil {
			return err
		}
		chunkText = string(textBucket.Get(int64ToBytes(nextChunk)))
		return nil
	})
	return chunkText, err
}

func getTexts(b *bolt.Bucket, id []byte) (texts UserTexts, err error) {
	v := b.Get(id)
	if v == nil {
		return defaultUserTexts(), nil
	}
	err = json.Unmarshal(v, &texts)
	if err != nil {
		return defaultUserTexts(), err
	}
	return texts, nil
}

func putTexts(b *bolt.Bucket, id []byte, texts UserTexts) error {
	encoded, err := json.Marshal(texts)
	if err != nil {
		return err
	}
	return b.Put(id, encoded)
}

func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func bytesToInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func defaultUserTexts() UserTexts {
	return UserTexts{
		Texts:   []Text{},
		Current: NotSelected,
	}
}

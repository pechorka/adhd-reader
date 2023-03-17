package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("not found")

const NotSelected = -1

var (
	bktUserInfo = []byte("user_info")
)

var (
	fullTextKey     = []byte("full_text")
	currentChunkKey = []byte("current_chunk")
	totalChunksKey  = []byte("total_chunks")
)

// Storage is a wrapper around bolt.DB
type Storage struct {
	db        *bolt.DB
	closeFunc func() error
}

// NewStorage creates a new storage
func NewStorage(path string) (*Storage, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Storage{
		db:        db,
		closeFunc: db.Close,
	}, nil
}

func NewTempStorage() (*Storage, error) {
	path := fmt.Sprintf("/tmp/%s.db", uuid.New().String())
	storage, err := NewStorage(path)
	if err != nil {
		return nil, err
	}
	originalCloseFunc := storage.closeFunc
	storage.closeFunc = func() error {
		if err := originalCloseFunc(); err != nil {
			return err
		}
		return os.Remove(path)
	}
	return storage, nil
}

// Close closes the storage
func (s *Storage) Close() error {
	return s.closeFunc()
}

func (s *Storage) AddText(userID int64, newText NewText) (string, error) {
	textUUID := uuid.New().String()
	err := s.db.Update(func(tx *bolt.Tx) error {
		// update user bucket
		b, err := tx.CreateBucketIfNotExists(bktUserInfo)
		if err != nil {
			return err
		}
		id := textsId(userID)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		for _, text := range texts.Texts {
			if text.Name == newText.Name {
				return fmt.Errorf("text with name %q already exists", newText.Name)
			}
		}
		textBucketName := []byte(uuid.New().String())
		texts.Texts = append(texts.Texts, Text{
			UUID:       textUUID,
			Name:       newText.Name,
			BucketName: textBucketName,
		})
		if err = putTexts(b, id, texts); err != nil {
			return err
		}
		// fill the text bucket
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
		return nil
	})
	return textUUID, err
}

func (s *Storage) GetTexts(id int64) (UserTexts, error) {
	var texts UserTexts
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return nil
		}
		var err error
		texts, err = getTexts(b, textsId(id))
		return err
	})
	return texts, err
}

type UpdateTextsFunc func(*UserTexts) error

func (s *Storage) UpdateTexts(userID int64, updFunc UpdateTextsFunc) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktUserInfo)
		if err != nil {
			return err
		}
		id := textsId(userID)
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

type SelectChunkFunc func(text Text, curChunk, totalChunks int64) (nextChunk int64, err error)

func (s *Storage) SelectChunk(userID int64, updFunc SelectChunkFunc) (string, error) {
	var chunkText string
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktUserInfo)
		if err != nil {
			return err
		}
		id := textsId(userID)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		if texts.Current == NotSelected {
			return errors.New("no text selected")
		}
		curText := texts.Texts[texts.Current]
		textBucket := tx.Bucket(curText.BucketName)
		if textBucket == nil { // should not happen
			return errors.New("unexpected error: text bucket not found")
		}
		curChunk := bytesToInt64(textBucket.Get(currentChunkKey))
		totalChunks := bytesToInt64(textBucket.Get(totalChunksKey))
		nextChunk, err := updFunc(curText, curChunk, totalChunks)
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

func (s *Storage) GetChunkSize(userID int64) (int64, error) {
	var chunkSize int64
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return nil
		}
		id := chunkSizeId(userID)
		chunkSize = getChunkSize(b, id)
		return nil
	})
	return chunkSize, err
}

func (s *Storage) SetChunkSize(userID int64, chunkSize int64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktUserInfo)
		if err != nil {
			return err
		}
		id := chunkSizeId(userID)
		return putChunkSize(b, id, chunkSize)
	})
}

func (s *Storage) DeleteTextByUUID(userID int64, textUUID string) error {
	return s.deleteTextBy(userID, func(text Text) bool {
		return text.UUID == textUUID
	})
}

func (s *Storage) DeleteTextByName(userID int64, textName string) error {
	return s.deleteTextBy(userID, func(text Text) bool {
		return text.Name == textName
	})
}

func (s *Storage) deleteTextBy(userID int64, predicate func(Text) bool) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return ErrNotFound
		}
		id := textsId(userID)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		var found bool
		for i, text := range texts.Texts {
			if predicate(text) {
				if err = tx.DeleteBucket(text.BucketName); err != nil {
					return err
				}
				texts.Texts = append(texts.Texts[:i], texts.Texts[i+1:]...)
				if texts.Current == i {
					texts.Current = NotSelected
				}
				found = true
				break
			}
		}
		if !found {
			return ErrNotFound
		}
		return putTexts(b, id, texts)
	})
}

// helper functions

func textsId(id int64) []byte {
	return []byte(fmt.Sprintf("texts-%d", id))
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

func chunkSizeId(id int64) []byte {
	return []byte(fmt.Sprintf("chunk-size-%d", id))
}

func getChunkSize(b *bolt.Bucket, id []byte) (size int64) {
	v := b.Get(id)
	if v == nil {
		return 0
	}
	return bytesToInt64(v)
}

func putChunkSize(b *bolt.Bucket, id []byte, size int64) error {
	return b.Put(id, int64ToBytes(size))
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

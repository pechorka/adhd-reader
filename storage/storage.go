package storage

import (
	"encoding/binary"
	"encoding/json"

	bolt "go.etcd.io/bbolt"
)

var (
	bktTexts = []byte("texts")
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

func (s *Storage) GetTexts(id int64) (*UserTexts, error) {
	var texts *UserTexts
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

func (s *Storage) PutText(userID int64, texts *UserTexts) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktTexts)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		return putTexts(b, id, texts)
	})
}

func getTexts(b *bolt.Bucket, id []byte) (*UserTexts, error) {
	v := b.Get(id)
	if v == nil {
		return nil, nil
	}
	var texts UserTexts
	err := json.Unmarshal(v, &texts)
	if err != nil {
		return nil, err
	}
	return &texts, nil
}

func putTexts(b *bolt.Bucket, id []byte, texts *UserTexts) error {
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

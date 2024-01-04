package chunks

import (
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var (
	ErrNotFound = errors.New("not found")
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
	return &Storage{
		db: db,
	}, nil
}

// Close closes the storage
func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) AddText(text string, chunks []string) (string, error) {
	textUUID := uuid.New().String()
	err := s.db.Update(func(tx *bolt.Tx) error {
		textBucket, err := tx.CreateBucketIfNotExists([]byte(textUUID))
		if err != nil {
			return err
		}
		if err = textBucket.Put([]byte(`fullText`), []byte(text)); err != nil { // for gracefull migrations later
			return err
		}
		for i, chunk := range chunks {
			if err = textBucket.Put(int64ToBytes(int64(i)), []byte(chunk)); err != nil {
				return err
			}
		}
		return nil
	})
	return textUUID, err
}

func (s *Storage) GetChunk(textUUID string, chunkIndex int) (string, error) {
	var chunk string
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(textUUID))
		if bkt == nil {
			return ErrNotFound
		}

		rawChunk := bkt.Get(int64ToBytes(int64(chunkIndex)))
		if rawChunk == nil {
			return ErrNotFound
		}

		chunk = string(rawChunk)

		return nil
	})

	return chunk, err
}

func (s *Storage) DeleteTextByUUID(textUUID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(textUUID))
	})
}

// helper functions

func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

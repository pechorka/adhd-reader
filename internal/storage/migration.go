package storage

import (
	"bytes"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

// MigrateCurrentChunk migrates current chunk from text bucket to user bucket
func (s *Storage) MigrateCurrentChunk() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return nil
		}
		currentChunkKey := []byte("current_chunk")
		return b.ForEach(func(k, v []byte) error {
			if !bytes.HasPrefix(k, textsPrefix) {
				return nil
			}
			texts, err := unmarshalTexts(v)
			if err != nil {
				return err
			}
			var skipped int = 0
			for i, text := range texts.Texts {
				if text.CurrentChunk != nil {
					skipped++
					continue // already migrated
				}
				textBucket := tx.Bucket(text.BucketName)
				if textBucket == nil {
					continue
				}
				currentChunk := bytesToInt64(textBucket.Get(currentChunkKey))
				text.CurrentChunk = &currentChunk
				texts.Texts[i] = text
				err = textBucket.Delete(currentChunkKey)
				if err != nil {
					return errors.Wrap(err, "failed to delete current chunk key")
				}
			}
			if skipped == len(texts.Texts) {
				return nil
			}
			err = putTexts(b, k, texts)
			if err != nil {
				return errors.Wrap(err, "failed to put texts")
			}
			return nil
		})
	})
}

package storage

import (
	"encoding/json"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

func (s *Storage) DeleteProcessedFilesWithNonexistentBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bktProcessedFiles)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var pf ProcessedFile
			if err := json.Unmarshal(v, &pf); err != nil {
				return errors.Wrap(err, "failed to unmarshal processed file")
			}
			if tx.Bucket(pf.BucketName) == nil {
				err := bucket.Delete(k)
				if err != nil {
					return errors.Wrap(err, "failed to delete processed file from bucket")
				}
			}
			return nil
		})
	})
}

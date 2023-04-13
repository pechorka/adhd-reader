package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("not found")

const NotSelected = -1

var (
	bktUserInfo       = []byte("user_info")
	bktProcessedFiles = []byte("processed_files")
)

var (
	fullTextKey    = []byte("full_text")
	totalChunksKey = []byte("total_chunks")
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
		if err = validateUserTexts(texts, textNameUnique(newText.Name)); err != nil {
			return err
		}
		textBucketName, err := fillTextBucket(tx, newText.Text, newText.Chunks)
		if err != nil {
			return err
		}
		texts.Texts = append(texts.Texts, Text{
			UUID:         textUUID,
			Name:         newText.Name,
			Source:       SourceText,
			BucketName:   textBucketName,
			CurrentChunk: NotSelected,
		})
		if err = putTexts(b, id, texts); err != nil {
			return err
		}
		return nil
	})
	return textUUID, err
}

func (s *Storage) AddTextFromProcessedFile(userId int64, name string, pf ProcessedFile) (string, error) {
	return pf.UUID, s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktUserInfo)
		if err != nil {
			return err
		}
		id := textsId(userId)
		texts, err := getTexts(b, id)
		if err != nil {
			return err
		}
		if err = validateUserTexts(
			texts,
			textNameUnique(name),
			textUUIDUnique(pf.UUID),
		); err != nil {
			return err
		}
		texts.Texts = append(texts.Texts, Text{
			UUID:         pf.UUID,
			Name:         name,
			Source:       SourceFile,
			BucketName:   pf.BucketName,
			CurrentChunk: NotSelected,
		})
		if err = putTexts(b, id, texts); err != nil {
			return err
		}
		return nil
	})
}

type textValidatorFunc func(texts Text) error

func validateUserTexts(texts UserTexts, validators ...textValidatorFunc) error {
	for _, text := range texts.Texts {
		for _, validator := range validators {
			if err := validator(text); err != nil {
				return err
			}
		}
	}
	return nil
}

func textNameUnique(textName string) textValidatorFunc {
	return func(text Text) error {
		if text.Name == textName {
			return fmt.Errorf("text with name %q already exists", textName)
		}
		return nil
	}
}

type TextAlreadyExistsError struct {
	ExistingText Text
}

func (e *TextAlreadyExistsError) Error() string {
	return fmt.Sprintf("text already exists by the name: %s", e.ExistingText.Name)
}

func textUUIDUnique(textUUID string) textValidatorFunc {
	return func(text Text) error {
		if text.UUID == textUUID {
			return &TextAlreadyExistsError{ExistingText: text}
		}
		return nil
	}
}

func (s *Storage) GetTexts(id int64) ([]TextWithChunkInfo, error) {
	var result []TextWithChunkInfo
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return nil
		}
		var err error
		texts, err := getTexts(b, textsId(id))
		if err != nil {
			return err
		}
		result, err = enrichTexts(tx, texts)
		return err
	})
	return result, err
}

func (s *Storage) GetFullTexts(id int64) ([]FullText, error) {
	var result []FullText
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return nil
		}
		var err error
		texts, err := getTexts(b, textsId(id))
		if err != nil {
			return err
		}
		result, err = fullTexts(tx, texts)
		return err
	})
	return result, err
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
		totalChunks := bytesToInt64(textBucket.Get(totalChunksKey))
		nextChunk, err := updFunc(curText, curText.CurrentChunk, totalChunks)
		if err != nil {
			return err
		}
		curText.CurrentChunk = nextChunk
		texts.Texts[texts.Current] = curText
		if err = putTexts(b, id, texts); err != nil {
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
				// texts from file share the same bucket between users
				if text.Source != SourceFile {
					if err = tx.DeleteBucket(text.BucketName); err != nil && err != bolt.ErrBucketNotFound {
						return err
					}
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

func (s *Storage) AddProcessedFile(newPf NewProcessedFile) (ProcessedFile, error) {
	var pf ProcessedFile
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktProcessedFiles)
		if err != nil {
			return err
		}
		textBucketName, err := fillTextBucket(tx, newPf.Text, newPf.Chunks)
		if err != nil {
			return err
		}
		pf = ProcessedFile{
			UUID:       uuid.NewString(),
			BucketName: textBucketName,
			ChunkSize:  newPf.ChunkSize,
			CheckSum:   newPf.CheckSum,
		}
		return putProcessedFile(b, pf)
	})
	return pf, err
}

func (s *Storage) GetProcessedFileByChecksum(checksum []byte) (ProcessedFile, error) {
	var pf ProcessedFile
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktProcessedFiles)
		if b == nil {
			return ErrNotFound
		}
		var err error
		pf, err = getProcessedFile(b, checksum)
		return err
	})
	return pf, err
}

func (s *Storage) Analytics() ([]UserAnalytics, error) {
	var result []UserAnalytics
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserInfo)
		if b == nil {
			return nil
		}
		userChunkSize := make(map[string]int64)
		userTexts := make(map[string]UserTexts)
		err := b.ForEach(func(k, v []byte) error {
			switch {
			case bytes.HasPrefix(k, chunkSizePrefix):
				userID := string(k[len(chunkSizePrefix):])
				userChunkSize[userID] = bytesToInt64(v)
			case bytes.HasPrefix(k, textsPrefix):
				userID := string(k[len(textsPrefix):])
				texts, err := getTexts(b, k)
				if err != nil {
					return err
				}
				userTexts[userID] = texts
			}
			return nil
		})
		if err != nil {
			return err
		}

		result = make([]UserAnalytics, 0, len(userChunkSize))
		for strUserID, texts := range userTexts {
			userID, err := strconv.ParseInt(strUserID, 10, 64)
			if err != nil { // should not happen
				return errors.Wrap(err, "failed to parse user id")
			}
			textsAnalytics, err := enrichTexts(tx, texts)
			if err != nil {
				return errors.Wrap(err, "failed to enrich texts")
			}
			result = append(result, UserAnalytics{
				UserID:         userID,
				ChunkSize:      userChunkSize[strUserID],
				TotalTextCount: int64(len(texts.Texts)),
				CurrentText:    texts.Current,
				Texts:          textsAnalytics,
			})
		}
		return nil
	})
	return result, err
}

// helper functions

var textsPrefix = []byte("texts-")

func textsId(id int64) []byte {
	return []byte(fmt.Sprintf("%s%d", textsPrefix, id))
}

func getTexts(b *bolt.Bucket, id []byte) (texts UserTexts, err error) {
	v := b.Get(id)
	if v == nil {
		return defaultUserTexts(), nil
	}
	return unmarshalTexts(v)
}

func unmarshalTexts(v []byte) (texts UserTexts, err error) {
	err = json.Unmarshal(v, &texts)
	if err != nil {
		return defaultUserTexts(), errors.Wrap(err, "failed to unmarshal texts")
	}
	return texts, nil
}

// enrichTexts enriches texts with current chunk
func enrichTexts(tx *bolt.Tx, texts UserTexts) ([]TextWithChunkInfo, error) {
	result := make([]TextWithChunkInfo, 0, len(texts.Texts))
	for _, text := range texts.Texts {
		textBucket := tx.Bucket(text.BucketName)
		if textBucket == nil {
			return nil, errors.New("unexpected error: text bucket not found")
		}
		totalChunks := bytesToInt64(textBucket.Get(totalChunksKey))
		result = append(result, TextWithChunkInfo{
			UUID:         text.UUID,
			Name:         text.Name,
			CurrentChunk: text.CurrentChunk,
			TotalChunks:  totalChunks,
		})
	}
	return result, nil
}

func fullTexts(tx *bolt.Tx, texts UserTexts) ([]FullText, error) {
	result := make([]FullText, 0, len(texts.Texts))
	for _, text := range texts.Texts {
		textBucket := tx.Bucket(text.BucketName)
		if textBucket == nil {
			return nil, errors.New("unexpected error: text bucket not found")
		}
		totalChunks := bytesToInt64(textBucket.Get(totalChunksKey))
		chunks := make([]string, 0, totalChunks)
		for i := int64(0); i < totalChunks; i++ {
			chunks = append(chunks, string(textBucket.Get(int64ToBytes(i))))
		}
		result = append(result, FullText{
			UUID:         text.UUID,
			Name:         text.Name,
			CurrentChunk: text.CurrentChunk,
			Chunks:       chunks,
		})
	}
	return result, nil
}

func putTexts(b *bolt.Bucket, id []byte, texts UserTexts) error {
	encoded, err := json.Marshal(texts)
	if err != nil {
		return err
	}
	return b.Put(id, encoded)
}

var chunkSizePrefix = []byte("chunk-size-")

func chunkSizeId(id int64) []byte {
	return []byte(fmt.Sprintf("%s%d", chunkSizePrefix, id))
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

func putProcessedFile(b *bolt.Bucket, pf ProcessedFile) error {
	encoded, err := json.Marshal(pf)
	if err != nil {
		return err
	}
	return b.Put(pf.CheckSum, encoded)
}

func getProcessedFile(b *bolt.Bucket, checksum []byte) (pf ProcessedFile, err error) {
	v := b.Get(checksum)
	if v == nil {
		return pf, ErrNotFound
	}
	err = json.Unmarshal(v, &pf)
	if err != nil {
		return pf, errors.Wrap(err, "failed to unmarshal processed file")
	}
	return pf, nil
}

func fillTextBucket(tx *bolt.Tx, text string, chunks []string) ([]byte, error) {
	textBucketName := []byte(uuid.New().String())
	textBucket, err := tx.CreateBucketIfNotExists(textBucketName)
	if err != nil {
		return nil, err
	}
	if err = textBucket.Put(fullTextKey, []byte(text)); err != nil {
		return nil, err
	}
	totalChunks := int64(len(chunks))
	if err = textBucket.Put(totalChunksKey, int64ToBytes(totalChunks)); err != nil {
		return nil, err
	}
	for i, chunk := range chunks {
		if err = textBucket.Put(int64ToBytes(int64(i)), []byte(chunk)); err != nil {
			return nil, err
		}
	}
	return textBucketName, nil
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

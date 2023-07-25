package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

const NotSelected = -1

var (
	bktUserInfo       = []byte("user_info")
	bktProcessedFiles = []byte("processed_files")
	bktDust           = []byte("dust")
	bktHerb           = []byte("herb")
	bktLevel          = []byte("level")
	bktStat           = []byte("stat")
	bktRecipe         = []byte("recipe")
	bktUserRecipe     = []byte("user_recipe")
	bktAuth           = []byte("auth")
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
	path := filepath.Join(os.TempDir(), fmt.Sprintf("adhd-reader-%s.db", uuid.New().String()))
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
		now := time.Now()
		texts.Texts = append(texts.Texts, Text{
			UUID:         textUUID,
			Name:         newText.Name,
			Source:       SourceText,
			BucketName:   textBucketName,
			CurrentChunk: NotSelected,
			CreatedAt:    now,
			ModifiedAt:   now,
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
		now := time.Now()
		texts.Texts = append(texts.Texts, Text{
			UUID:         pf.UUID,
			Name:         name,
			Source:       SourceFile,
			BucketName:   pf.BucketName,
			CurrentChunk: NotSelected,
			CreatedAt:    now,
			ModifiedAt:   now,
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
		curText.ModifiedAt = time.Now()
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

func (s *Storage) UpdateDust(userID int64, updFunc func(*Dust)) (*Dust, error) {
	var dust Dust
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktDust)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		dust, err = s.getDust(b, id)
		if err != nil {
			return err
		}
		updFunc(&dust)
		return s.putDust(b, id, dust)
	})
	return &dust, err
}

func (s *Storage) GetDustByUserID(userID int64) (dust Dust, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktDust)
		if b == nil {
			return nil
		}
		id := int64ToBytes(userID)
		dust, err = s.getDust(b, id)
		return err
	})
	return dust, err
}

func (s *Storage) UpdateHerb(userID int64, updFunc func(*Herb)) (*Herb, error) {
	var herb Herb
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktHerb)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		herb, err = s.getHerb(b, id)
		if err != nil {
			return err
		}
		updFunc(&herb)
		return s.putHerb(b, id, herb)
	})
	return &herb, err
}

func (s *Storage) GetHerbByUserID(userID int64) (herb Herb, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktHerb)
		if b == nil {
			return nil
		}
		id := int64ToBytes(userID)
		herb, err = s.getHerb(b, id)
		return err
	})
	return herb, err
}

func (s *Storage) UpdateLevel(userID int64, updFunc func(*Level)) (*Level, error) {
	var level Level
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktLevel)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		level, err = s.getLevel(b, id)
		if err != nil {
			return err
		}
		updFunc(&level)
		return s.putLevel(b, id, level)
	})
	return &level, err
}

func (s *Storage) GetLevelByUserID(userID int64) (level Level, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktLevel)
		if b == nil {
			return nil
		}
		id := int64ToBytes(userID)
		level, err = s.getLevel(b, id)
		return err
	})
	return level, err
}

func (s *Storage) UpdateStat(userID int64, updFunc func(*Stat)) (*Stat, error) {
	var stat Stat
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktStat)
		if err != nil {
			return err
		}
		id := int64ToBytes(userID)
		stat, err = s.getStat(b, id)
		if err != nil {
			return err
		}
		updFunc(&stat)
		return s.putStat(b, id, stat)
	})
	return &stat, err
}

func (s *Storage) GetStatByUserID(userID int64) (stat Stat, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktStat)
		if b == nil {
			return nil
		}
		id := int64ToBytes(userID)
		stat, err = s.getStat(b, id)
		return err
	})
	return stat, err
}

func (s *Storage) GetRecipeByName(name string) (recipe Recipe, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktRecipe)
		if b == nil {
			return nil
		}
		recipe, err = s.getRecipe(b, []byte(name))
		return err
	})
	return recipe, err
}

func (s *Storage) GetUserRecipeByUserIDandRecipeName(userID int64, recipeName string) (recipe UserRecipe, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktUserRecipe)
		if b == nil {
			return nil
		}
		recipe, err = s.getUserRecipe(b, userID, recipeName)
		return err
	})
	return recipe, err
}

func (s *Storage) UpdateUserRecipe(userID int64, recipeName string, updFunc func(*UserRecipe)) (*UserRecipe, error) {
	var userRecipe UserRecipe
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktUserRecipe)
		if err != nil {
			return err
		}
		userRecipe, err = s.getUserRecipe(b, userID, recipeName)
		if err != nil {
			return err
		}
		updFunc(&userRecipe)
		return s.putUserRecipe(b, int64ToBytes(userID), userRecipe)
	})
	return &userRecipe, err
}

func (s *Storage) SetAuthToken(userID int64, token string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bktAuth)
		if err != nil {
			return err
		}
		byteUserID := int64ToBytes(userID)
		if b.Get(byteUserID) != nil {
			return ErrAlreadyExists
		}
		byteToken := []byte(token)
		if dbUserID := b.Get(byteToken); dbUserID != nil {
			if bytes.Equal(dbUserID, byteUserID) {
				return nil
			}
			return ErrAlreadyExists
		}
		if err = b.Put(byteToken, byteUserID); err != nil {
			return errors.Wrap(err, "failed to put user id by token")
		}
		if err = b.Put(byteUserID, byteToken); err != nil {
			return errors.Wrap(err, "failed to put token by user id")
		}
		return nil
	})
}

func (s *Storage) GetUserIDByAuthToken(token string) (int64, error) {
	var userID int64
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktAuth)
		if b == nil {
			return ErrNotFound
		}
		byteUserID := b.Get([]byte(token))
		if byteUserID == nil {
			return ErrNotFound
		}
		userID = bytesToInt64(byteUserID)
		return nil
	})
	return userID, err
}

func (s *Storage) GetTokenByUserID(userID int64) (string, error) {
	var token string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktAuth)
		if b == nil {
			return ErrNotFound
		}
		byteToken := b.Get(int64ToBytes(userID))
		if byteToken == nil {
			return ErrNotFound
		}
		token = string(byteToken)
		return nil
	})
	return token, err
}

func (s *Storage) DeleteAuthToken(userID int64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bktAuth)
		if b == nil {
			return ErrNotFound
		}
		byteUserID := int64ToBytes(userID)
		byteToken := b.Get(byteUserID)
		if byteToken == nil {
			return ErrNotFound
		}
		if err := b.Delete(byteToken); err != nil {
			return errors.Wrap(err, "failed to delete token by user id")
		}
		if err := b.Delete(byteUserID); err != nil {
			return errors.Wrap(err, "failed to delete user id by token")
		}
		return nil
	})
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

func (s *Storage) getDust(b *bolt.Bucket, id []byte) (dust Dust, err error) {
	v := b.Get(id)
	if v == nil {
		return dust, nil
	}
	err = json.Unmarshal(v, &dust)
	if err != nil {
		return dust, errors.Wrap(err, "failed to unmarshal dust")
	}
	return dust, nil
}

func (s *Storage) putDust(b *bolt.Bucket, id []byte, dust Dust) error {
	encoded, err := json.Marshal(dust)
	if err != nil {
		return err
	}
	return b.Put(id, encoded)
}

func (s *Storage) getHerb(b *bolt.Bucket, id []byte) (herb Herb, err error) {
	v := b.Get(id)
	if v == nil {
		return herb, nil
	}
	err = json.Unmarshal(v, &herb)
	if err != nil {
		return herb, errors.Wrap(err, "failed to unmarshal herb")
	}
	return herb, nil
}

func (s *Storage) putHerb(b *bolt.Bucket, id []byte, herb Herb) error {
	encoded, err := json.Marshal(herb)
	if err != nil {
		return err
	}
	return b.Put(id, encoded)
}

func (s *Storage) getLevel(b *bolt.Bucket, id []byte) (level Level, err error) {
	v := b.Get(id)
	if v == nil {
		return level, nil
	}
	err = json.Unmarshal(v, &level)
	if err != nil {
		return level, errors.Wrap(err, "failed to unmarshal level")
	}
	return level, nil
}

func (s *Storage) putLevel(b *bolt.Bucket, id []byte, level Level) error {
	encoded, err := json.Marshal(level)
	if err != nil {
		return err
	}
	return b.Put(id, encoded)
}

func (s *Storage) getStat(b *bolt.Bucket, id []byte) (stat Stat, err error) {
	v := b.Get(id)
	if v == nil {
		return stat, nil
	}
	err = json.Unmarshal(v, &stat)
	if err != nil {
		return stat, errors.Wrap(err, "failed to unmarshal stat")
	}
	return stat, nil
}

func (s *Storage) putStat(b *bolt.Bucket, id []byte, stat Stat) error {
	encoded, err := json.Marshal(stat)
	if err != nil {
		return err
	}
	return b.Put(id, encoded)
}

func (s *Storage) getRecipe(b *bolt.Bucket, id []byte) (recipe Recipe, err error) {
	v := b.Get(id)
	if v == nil {
		return recipe, nil
	}
	err = json.Unmarshal(v, &recipe)
	if err != nil {
		return recipe, errors.Wrap(err, "failed to unmarshal stat")
	}
	return recipe, nil
}

func (s *Storage) getUserRecipe(b *bolt.Bucket, userID int64, recipeName string) (userRecipe UserRecipe, err error) {
	// Create the key by concatenating userID and recipeName
	key := fmt.Sprintf("%s|%s", strconv.FormatInt(userID, 10), recipeName)

	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MyBucket"))
		v := b.Get([]byte(key))

		if v == nil {
			return errors.New("No value found for this key")
		}

		err := json.Unmarshal(v, &userRecipe)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal user recipe")
		}
		return nil
	})

	if err != nil {
		return UserRecipe{}, errors.Wrap(err, "Failed to get user recipe: %v")
	}

	return userRecipe, nil
}

func (s *Storage) putUserRecipe(b *bolt.Bucket, id []byte, recipe UserRecipe) error {
	encoded, err := json.Marshal(recipe)
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

package service

import (
	"unicode/utf8"

	"github.com/pechorka/adhd-reader/internal/storage"

	"github.com/pechorka/adhd-reader/pkg/textspliter"
	"github.com/pkg/errors"
)

var ErrTextFinished = errors.New("text finished")
var ErrFirstChunk = errors.New("first chunk")
var ErrTextNotSelected = errors.New("text is not selected")
var ErrTextNotUTF8 = errors.New("text is not valid utf8")

const telegramMessageLengthLimit = 4096

type Service struct {
	s         *storage.Storage
	chunkSize int64
}

func NewService(s *storage.Storage, chunkSize int64) *Service {
	return &Service{s: s, chunkSize: chunkSize}
}

func (s *Service) SetChunkSize(userID int64, chunkSize int64) error {
	if chunkSize < 1 {
		return errors.New("chunk size must be greater than 0")
	}
	if chunkSize > telegramMessageLengthLimit {
		return errors.Errorf("chunk size is too big, telegram message length limit is %d", telegramMessageLengthLimit)
	}
	return s.s.SetChunkSize(userID, chunkSize)
}

func (s *Service) AddText(userID int64, textName, text string) (string, error) {
	if textName == "" {
		return "", errors.New("text name is empty")
	}
	if len(textName) > 255 {
		return "", errors.Errorf("text name %s is too long, max length is 255 (less if you use emojis/non-ascii symbols)", textName)
	}
	if !utf8.ValidString(text) {
		return "", ErrTextNotUTF8
	}
	chunkSize, err := s.s.GetChunkSize(userID)
	if err != nil {
		return "", err
	}
	if chunkSize == 0 {
		chunkSize = s.chunkSize
	}
	textChunks := textspliter.SplitText(text, int(chunkSize))
	data := storage.NewText{
		Name:      textName,
		Chunks:    textChunks,
		Text:      text,
		ChunkSize: chunkSize,
	}
	return s.s.AddText(userID, data)
}

type TextWithCompletion struct {
	UUID              string
	Name              string
	CompletionPercent int
}

func (s *Service) ListTexts(userID int64) ([]TextWithCompletion, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return nil, err
	}
	result := make([]TextWithCompletion, 0, len(texts))
	for _, t := range texts {
		result = append(result, TextWithCompletion{
			UUID:              t.UUID,
			Name:              t.Name,
			CompletionPercent: calculateCompletionPercent(t),
		})
	}
	return result, nil
}

func calculateCompletionPercent(text storage.TextWithChunkInfo) int {
	if text.TotalChunks-1 <= 0 || text.CurrentChunk == storage.NotSelected {
		return 0
	}
	return int(float64(text.CurrentChunk) / float64(text.TotalChunks-1) * 100)
}

func (s *Service) SelectText(userID int64, textUUID string) (storage.Text, error) {
	var text storage.Text
	err := s.s.UpdateTexts(userID, func(texts *storage.UserTexts) error {
		for i, t := range texts.Texts {
			if t.UUID == textUUID {
				texts.Current = i
				text = t
				return nil
			}
		}
		return errors.Errorf("text with uuid %s not found", textUUID)
	})
	return text, err
}

func (s *Service) SetPage(userID, page int64) error {
	_, err := s.s.SelectChunk(userID, func(_ storage.Text, _, totalChunks int64) (nextChunk int64, err error) {
		if page >= totalChunks || page < 0 {
			return 0, errors.Errorf("invalid page index, should be between 0 and %d", totalChunks-1)
		}
		return page, nil
	})
	return err
}

func (s *Service) NextChunk(userID int64) (storage.Text, string, ChunkType, error) {
	return s.selectChunk(userID, func(_ storage.Text, curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk >= totalChunks-1 {
			return 0, ErrTextFinished
		}
		return curChunk + 1, nil
	})
}

func (s *Service) PrevChunk(userID int64) (storage.Text, string, ChunkType, error) {
	return s.selectChunk(userID, func(_ storage.Text, curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk <= 0 {
			return 0, ErrFirstChunk
		}
		return curChunk - 1, nil
	})
}

func (s *Service) CurrentOrFirstChunk(userID int64) (storage.Text, string, ChunkType, error) {
	return s.selectChunk(userID, func(_ storage.Text, curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk == storage.NotSelected {
			return 0, nil // return first chunk
		}
		return curChunk, nil
	})
}

type ChunkType string

const (
	ChunkTypeFirst ChunkType = "first"
	ChunkTypeLast  ChunkType = "last"
)

func (s *Service) selectChunk(userID int64, selectChunk storage.SelectChunkFunc) (storage.Text, string, ChunkType, error) {
	var chunkType ChunkType
	var curText storage.Text
	text, err := s.s.SelectChunk(userID, func(text storage.Text, curChunk, totalChunks int64) (nextChunk int64, err error) {
		curText = text
		nextChunk, err = selectChunk(text, curChunk, totalChunks)
		if err != nil {
			return 0, err
		}
		if nextChunk == 0 {
			chunkType = ChunkTypeFirst
		}
		if nextChunk == totalChunks-1 {
			chunkType = ChunkTypeLast
		}
		return nextChunk, nil
	})
	return curText, text, chunkType, err
}

func (s *Service) DeleteTextByUUID(userID int64, textUUID string) error {
	return s.s.DeleteTextByUUID(userID, textUUID)
}

func (s *Service) DeleteTextByName(userID int64, textUUID string) error {
	return s.s.DeleteTextByName(userID, textUUID)
}

type UserAnalytics struct {
	UserID              int64
	ChunkSize           int64
	TotalTextCount      int64
	AvgTotalChunks      int64
	MaxCurrentChunk     int64
	StartedTextsCount   int64
	CompletedTextsCount int64
	CurrentTextName     string
}

func (s *Service) Analytics() ([]UserAnalytics, error) {
	rawAnalytics, err := s.s.Analytics()
	if err != nil {
		return nil, err
	}
	result := make([]UserAnalytics, 0, len(rawAnalytics))
	for _, userAnalytics := range rawAnalytics {
		var totalChunks int64
		var maxCurrentChunk int64 = storage.NotSelected
		var startedTextsCount int64
		var completedTextsCount int64
		for _, text := range userAnalytics.Texts {
			totalChunks += text.TotalChunks
			if text.CurrentChunk > maxCurrentChunk {
				maxCurrentChunk = text.CurrentChunk
			}
			if text.CurrentChunk != storage.NotSelected {
				startedTextsCount++
			}
			if text.CurrentChunk == text.TotalChunks-1 {
				completedTextsCount++
			}
		}
		currentTextName := "Not selected"
		if userAnalytics.CurrentText != storage.NotSelected {
			currentTextName = userAnalytics.Texts[userAnalytics.CurrentText].Name
		}
		chunkSize := userAnalytics.ChunkSize
		if chunkSize == 0 {
			chunkSize = s.chunkSize
		}
		result = append(result, UserAnalytics{
			UserID:              userAnalytics.UserID,
			ChunkSize:           chunkSize,
			TotalTextCount:      userAnalytics.TotalTextCount,
			AvgTotalChunks:      totalChunks / int64(len(userAnalytics.Texts)),
			MaxCurrentChunk:     maxCurrentChunk,
			StartedTextsCount:   startedTextsCount,
			CompletedTextsCount: completedTextsCount,
			CurrentTextName:     currentTextName,
		})
	}
	return result, nil
}

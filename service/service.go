package service

import (
	"strings"

	"github.com/aakrasnova/zone-mate/storage"
	"github.com/pkg/errors"
)

var ErrTextFinished = errors.New("text finished")
var ErrFirstChunk = errors.New("first chunk")
var ErrTextNotSelected = errors.New("text is not selected")

type Service struct {
	s         *storage.Storage
	chunkSize int
}

func NewService(s *storage.Storage, chunkSize int) *Service {
	return &Service{s: s, chunkSize: chunkSize}
}

func (s *Service) AddText(userID int64, textName, text string) error {
	textChunks := splitText(text, s.chunkSize)
	data := storage.NewText{
		Name:   textName,
		Chunks: textChunks,
		Text:   text,
	}
	return s.s.AddText(userID, data)
}

func (s *Service) ListTexts(userID int64) ([]string, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, t := range texts.Texts {
		names = append(names, t.Name)
	}
	return names, nil
}

func (s *Service) SelectText(userID int64, current int) error {
	return s.s.UpdateTexts(userID, func(texts *storage.UserTexts) error {
		if current >= len(texts.Texts) || current < 0 {
			return errors.Errorf("invalid text index, should be between 0 and %d", len(texts.Texts)-1)
		}
		texts.Current = current
		return nil
	})
}

func (s *Service) SetPage(userID, page int64) error {
	_, err := s.s.SelectChunk(userID, func(curChunk, totalChunks int64) (nextChunk int64, err error) {
		if page >= totalChunks || page < 0 {
			return 0, errors.Errorf("invalid page index, should be between 0 and %d", totalChunks-1)
		}
		return page, nil
	})
	return err
}

func (s *Service) NextChunk(userID int64) (string, error) {
	return s.s.SelectChunk(userID, func(curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk >= totalChunks {
			return 0, ErrTextFinished
		}
		return curChunk + 1, nil
	})
}

func (s *Service) PrevChunk(userID int64) (string, error) {
	return s.s.SelectChunk(userID, func(curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk <= storage.NotSelected {
			return 0, ErrFirstChunk
		}
		return curChunk - 1, nil
	})
}

func splitText(text string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(text); {
		end := i + chunkSize
		for end < len(text)-1 {
			end++
			if endOfTheSentence(text[end]) {
				end++ // include space ender
				break
			}
		}
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, strings.TrimSpace(text[i:end]))
		i = end
	}
	return chunks
}

func endOfTheSentence(b byte) bool {
	return b == '.' || b == '!' || b == '?'
}

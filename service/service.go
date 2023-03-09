package service

import (
	"errors"

	"github.com/aakrasnova/zone-mate/storage"
)

var ErrTextFinished = errors.New("text finished")
var ErrFirstChunk = errors.New("first chunk")

type Service struct {
	s         *storage.Storage
	chunkSize int
}

func NewService(s *storage.Storage) *Service {
	return &Service{s: s, chunkSize: 200}
}

func (s *Service) AddText(userID int64, textName, text string) error {
	textChunks := splitText(text, s.chunkSize)
	data := storage.Text{
		Name:   textName,
		Chunks: textChunks,
	}
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return err
	}
	if texts == nil {
		texts = &storage.UserTexts{
			Texts: []storage.Text{data},
		}
	} else {
		texts.Texts = append(texts.Texts, data)
	}
	return s.s.PutText(userID, texts)
}

func (s *Service) ListTexts(userID int64) ([]string, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return nil, err
	}
	if texts == nil {
		return nil, nil
	}
	var names []string
	for _, t := range texts.Texts {
		names = append(names, t.Name)
	}
	return names, nil
}

func (s *Service) SelectText(userID int64, current int) error {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return err
	}
	if current >= len(texts.Texts) {
		return errors.New("invalid text index")
	}
	texts.Current = current
	return s.s.PutText(userID, texts)
}

func (s *Service) NextChunk(userID int64) (string, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return "", err
	}
	text := texts.Texts[texts.Current]
	text.LastRead++
	if text.LastRead >= len(text.Chunks) {
		return "", ErrTextFinished
	}
	chunk := text.Chunks[text.LastRead]
	texts.Texts[texts.Current] = text
	err = s.s.PutText(userID, texts)
	if err != nil {
		return "", err
	}
	return chunk, nil
}

func (s *Service) PrevChunk(userID int64) (string, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return "", err
	}
	text := texts.Texts[texts.Current]
	if text.LastRead <= 0 {
		return "", ErrFirstChunk
	}
	text.LastRead--
	chunk := text.Chunks[text.LastRead]
	texts.Texts[texts.Current] = text
	err = s.s.PutText(userID, texts)
	if err != nil {
		return "", err
	}
	return chunk, nil
}

func splitText(text string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}
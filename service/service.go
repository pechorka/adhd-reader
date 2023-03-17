package service

import (
	"strings"
	"unicode/utf8"

	"github.com/aakrasnova/zone-mate/storage"
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
	textChunks := splitText(text, int(chunkSize))
	data := storage.NewText{
		Name:      textName,
		Chunks:    textChunks,
		Text:      text,
		ChunkSize: chunkSize,
	}
	return s.s.AddText(userID, data)
}

func (s *Service) ListTexts(userID int64) ([]storage.Text, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return nil, err
	}
	return texts.Texts, nil
}

func (s *Service) CurrentText(userID int64) (storage.Text, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return storage.Text{}, err
	}
	if texts.Current == storage.NotSelected {
		return storage.Text{}, ErrTextNotSelected
	}
	return texts.Texts[texts.Current], nil
}

func (s *Service) SelectText(userID int64, textUUID string) error {
	return s.s.UpdateTexts(userID, func(texts *storage.UserTexts) error {
		for i, t := range texts.Texts {
			if t.UUID == textUUID {
				texts.Current = i
				return nil
			}
		}
		return errors.Errorf("text with uuid %s not found", textUUID)
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
		if curChunk >= totalChunks-1 {
			return 0, ErrTextFinished
		}
		return curChunk + 1, nil
	})
}

func (s *Service) PrevChunk(userID int64) (string, error) {
	return s.s.SelectChunk(userID, func(curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk <= 0 {
			return 0, ErrFirstChunk
		}
		return curChunk - 1, nil
	})
}

func (s *Service) CurrentOrNextChunk(userID int64) (string, error) {
	return s.s.SelectChunk(userID, func(curChunk, totalChunks int64) (nextChunk int64, err error) {
		if curChunk == storage.NotSelected {
			return 0, nil // return first chunk
		}
		return curChunk, nil
	})
}

func (s *Service) DeleteTextByUUID(userID int64, textUUID string) error {
	return s.s.DeleteTextByUUID(userID, textUUID)
}

func (s *Service) DeleteTextByName(userID int64, textUUID string) error {
	return s.s.DeleteTextByName(userID, textUUID)
}

func splitText(text string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(text); {
		end := i + chunkSize
		if end >= len(text) {
			end = len(text) - 1
		}
		// todo: handle telegram message length limit

		// backtracking to the nearest space to check if we are in the middle of the link
		var j int
		for j = end; j > i && text[j] != ' '; j-- {
		}
		if (text[j] == ' ' || j == i) && strings.HasPrefix(text[j+1:], "http") {
			// we are in the middle of the link, go until the end of the link
			for ; end < len(text) && text[end] != ' '; end++ {
			}
			if end >= len(text) {
				end = len(text) - 1
			}
		}
		// go until the end of the sentence
		for ; end < len(text); end++ {
			if endOfTheSentenceAt(text, end) {
				for endOfTheSentenceAt(text, end) { // skip multiple punctuation marks
					end++
				}
				if end >= len(text) {
					break
				}
				_, runeSize := utf8.DecodeRuneInString(text[end:])
				// skip i.e or т.д.
				if endOfTheSentenceAt(text, end+runeSize) {
					end += runeSize + 1 // +1 for the end of the sentence mark
					// at this point we could be in the middle of the sentence
					// or at the end of the sentence. We can't distinguish these cases.
					// It's ok to continue in either case, because
					// 1) if we are in the middle of the sentence, we need to find the end of the sentence
					// 2) if we are at the end of the sentence, it's ok to include another sentence in the chunk
					continue
				}
				break
			}
		}
		chunks = append(chunks, strings.TrimSpace(text[i:end]))
		i = end
	}
	return chunks
}

func endOfTheSentenceAt(text string, pos int) bool {
	if pos >= len(text) {
		return false
	}
	b := text[pos]
	return b == '.' || b == '!' || b == '?'
}

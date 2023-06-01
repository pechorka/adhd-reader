package service

import (
	"math/rand"
	"time"
	"unicode/utf8"

	"github.com/pechorka/adhd-reader/internal/storage"

	"github.com/pechorka/adhd-reader/pkg/chance"
	"github.com/pechorka/adhd-reader/pkg/textspliter"
	"github.com/pkg/errors"
)

var ErrTextFinished = errors.New("text finished")
var ErrFirstChunk = errors.New("first chunk")
var ErrTextNotSelected = errors.New("text is not selected")
var ErrTextNotUTF8 = errors.New("text is not valid utf8")

const telegramMessageLengthLimit = 4096

type Chancer interface {
	Win(percent float64) bool
	PickWin(inputs ...chance.WinInput)
}

type Service struct {
	s         *storage.Storage
	chancer   Chancer
	chunkSize int64
}

func NewService(s *storage.Storage, chunkSize int64) *Service {
	return &Service{s: s, chunkSize: chunkSize, chancer: chance.Default}
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
	chunkSize, err := s.getChunkSize(userID)
	if err != nil {
		return "", err
	}
	textChunks, err := s.processText(userID, textName, text, chunkSize)
	if err != nil {
		return "", err
	}
	data := storage.NewText{
		Name:      textName,
		Chunks:    textChunks,
		Text:      text,
		ChunkSize: chunkSize,
	}
	return s.s.AddText(userID, data)
}

func (s *Service) AddTextFromFile(userID int64, checksum []byte, name, text string) (string, error) {
	chunkSize, err := s.getChunkSize(userID)
	if err != nil {
		return "", err
	}
	pf, err := s.s.GetProcessedFileByChecksum(checksum)
	switch err {
	case nil:
		// can reuse processed file if chunk size is the same
		if pf.ChunkSize == chunkSize {
			return s.s.AddTextFromProcessedFile(userID, name, pf)
		}
	case storage.ErrNotFound:
	default:
		return "", err
	}

	textChunks, err := s.processText(userID, name, text, chunkSize)
	if err != nil {
		return "", err
	}

	pf, err = s.s.AddProcessedFile(storage.NewProcessedFile{
		Text:      text,
		Chunks:    textChunks,
		ChunkSize: chunkSize,
		CheckSum:  checksum,
	})
	if err != nil {
		return "", err
	}
	return s.s.AddTextFromProcessedFile(userID, name, pf)
}

func (s *Service) processText(userID int64, textName, text string, chunkSize int64) ([]string, error) {
	if textName == "" {
		return nil, errors.New("text name is empty")
	}
	if len(textName) > 255 {
		return nil, errors.Errorf("text name %s is too long, max length is 255 (less if you use emojis/non-ascii symbols)", textName)
	}
	if !utf8.ValidString(text) {
		return nil, ErrTextNotUTF8
	}
	return textspliter.SplitText(text, int(chunkSize)), nil
}

func (s *Service) getChunkSize(userID int64) (int64, error) {
	chunkSize, err := s.s.GetChunkSize(userID)
	if err != nil {
		return 0, err
	}
	if chunkSize == 0 {
		chunkSize = s.chunkSize
	}
	return chunkSize, nil
}

type TextWithCompletion struct {
	UUID              string
	Name              string
	CompletionPercent int
}

func (s *Service) ListTexts(userID int64, page, pageSize int) (_ []TextWithCompletion, more bool, err error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return nil, false, err
	}
	texts, more = paginateTexts(texts, page, pageSize)
	result := make([]TextWithCompletion, 0, len(texts))
	for _, t := range texts {
		result = append(result, TextWithCompletion{
			UUID:              t.UUID,
			Name:              t.Name,
			CompletionPercent: calculateCompletionPercent(t),
		})
	}
	return result, more, nil
}

func paginateTexts(texts []storage.TextWithChunkInfo, page, pageSize int) ([]storage.TextWithChunkInfo, bool) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(texts) {
		return nil, false
	}
	if end > len(texts) {
		end = len(texts)
	}
	return texts[start:end], end < len(texts)
}

func (s *Service) FullTexts(userID int64) ([]storage.FullText, error) {
	return s.s.GetFullTexts(userID)
}

func calculateCompletionPercent(text storage.TextWithChunkInfo) int {
	if text.TotalChunks-1 <= 0 || text.CurrentChunk == storage.NotSelected {
		return 0
	}
	return int(float64(text.CurrentChunk) / float64(text.TotalChunks-1) * 100)
}

func (s *Service) RandomText(userID int64, atMostChunks int64) (storage.TextWithChunkInfo, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return storage.TextWithChunkInfo{}, err
	}
	if atMostChunks > 0 {
		texts = filterTextsByChunkCount(texts, atMostChunks)
	}
	if len(texts) == 0 {
		return storage.TextWithChunkInfo{}, errors.New("no texts")
	}
	return texts[rand.Intn(len(texts))], nil
}

func filterTextsByChunkCount(texts []storage.TextWithChunkInfo, atMostChunks int64) []storage.TextWithChunkInfo {
	if atMostChunks <= 0 {
		return texts
	}
	result := make([]storage.TextWithChunkInfo, 0, len(texts))
	for _, t := range texts {
		if t.TotalChunks <= atMostChunks {
			result = append(result, t)
		}
	}
	return result
}

func (s *Service) SelectText(userID int64, textUUID string) (storage.Text, error) {
	var text storage.Text
	err := s.s.UpdateTexts(userID, func(texts *storage.UserTexts) error {
		for i, t := range texts.Texts {
			if t.UUID == textUUID {
				texts.Current = i
				texts.Texts[i].ModifiedAt = time.Now()
				text = t
				return nil
			}
		}
		return errors.Errorf("text with uuid %s not found", textUUID)
	})
	return text, err
}

func (s *Service) RenameText(userID int64, newName string) (string, error) {
	if newName == "" {
		return "", errors.New("text name is empty")
	}
	oldName := ""
	err := s.s.UpdateTexts(userID, func(texts *storage.UserTexts) error {
		if texts.Current == storage.NotSelected {
			return errors.New("no text selected")
		}
		for _, t := range texts.Texts {
			if t.Name == newName {
				return errors.Errorf("text with name %s already exists", newName)
			}
		}
		oldName = texts.Texts[texts.Current].Name
		texts.Texts[texts.Current].Name = newName
		texts.Texts[texts.Current].ModifiedAt = time.Now()
		return nil
	})
	return oldName, err
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

type TotalAnalytics struct {
	TotalNumberOfUsers     int64
	NumberOfUsersWithTexts int64
	TotalNumberOfTexts     int64
	AverageChunkSize       int64
}

func (s *Service) Analytics() ([]UserAnalytics, *TotalAnalytics, error) {
	rawAnalytics, err := s.s.Analytics()
	if err != nil {
		return nil, nil, err
	}
	result := make([]UserAnalytics, 0, len(rawAnalytics))
	var totalChunkSize int64
	var usersWithTexts int64
	var totalNumberOfTexts int64
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
		totalChunkSize += chunkSize
		if len(userAnalytics.Texts) > 0 {
			usersWithTexts++
		}
		totalNumberOfTexts += int64(len(userAnalytics.Texts))
		var avgTotalChunks int64
		if len(userAnalytics.Texts) > 0 {
			avgTotalChunks = totalChunks / int64(len(userAnalytics.Texts))
		}
		result = append(result, UserAnalytics{
			UserID:              userAnalytics.UserID,
			ChunkSize:           chunkSize,
			TotalTextCount:      userAnalytics.TotalTextCount,
			AvgTotalChunks:      avgTotalChunks,
			MaxCurrentChunk:     maxCurrentChunk,
			StartedTextsCount:   startedTextsCount,
			CompletedTextsCount: completedTextsCount,
			CurrentTextName:     currentTextName,
		})
	}
	numberOfUsers := int64(len(result))
	var averageChunkSize int64
	if numberOfUsers > 0 {
		averageChunkSize = totalChunkSize / numberOfUsers
	}

	totalAnalytics := &TotalAnalytics{
		TotalNumberOfUsers:     numberOfUsers,
		NumberOfUsersWithTexts: usersWithTexts,
		TotalNumberOfTexts:     totalNumberOfTexts,
		AverageChunkSize:       averageChunkSize,
	}
	return result, totalAnalytics, nil
}

type Dust struct {
	RedCount    int64
	OrangeCount int64
	YellowCount int64
	GreenCount  int64
	BlueCount   int64
	IndigoCount int64
	PurpleCount int64
	WhiteCount  int64
	BlackCount  int64
}

type Herb struct {
	LavandaCount int64
	MelissaCount int64
}

func (s *Service) LootOnNextChunk(userID int64) (*Dust, *Herb, Dust, Herb, error) {
	var deltaDust Dust
	var deltaHerb Herb
	// 1.9% chance to get Herb
	if s.chancer.Win(0.019) {
		s.chancer.PickWin(
			chance.WinInput{
				Percent: 0.7,
				Action: func() {
					deltaHerb.MelissaCount++
				},
			},
			chance.WinInput{
				Percent: 0.3,
				Action: func() {
					deltaHerb.LavandaCount++
				},
			},
		)
	}
	// 33% chance to get Dust
	if s.chancer.Win(0.33) {
		s.chancer.PickWin(
			chance.WinInput{
				Percent: 0.25,
				Action: func() {
					deltaDust.RedCount++
				},
			},
			chance.WinInput{
				Percent: 0.03,
				Action: func() {
					deltaDust.OrangeCount++
				},
			},
			chance.WinInput{
				Percent: 0.25,
				Action: func() {
					deltaDust.YellowCount++
				},
			},
			chance.WinInput{
				Percent: 0.142,
				Action: func() {
					deltaDust.GreenCount++
				},
			},
			chance.WinInput{
				Percent: 0.13,
				Action: func() {
					deltaDust.BlueCount++
				},
			},
			chance.WinInput{
				Percent: 0.13,
				Action: func() {
					deltaDust.IndigoCount++
				},
			},
			chance.WinInput{
				Percent: 0.03,
				Action: func() {
					deltaDust.PurpleCount++
				},
			},
			chance.WinInput{
				Percent: 0.019,
				Action: func() {
					deltaDust.WhiteCount++
				},
			},
			chance.WinInput{
				Percent: 0.019,
				Action: func() {
					deltaDust.BlackCount++
				},
			},
		)
	}

	dbDust, err := s.s.UpdateDust(userID, func(d *storage.Dust) {
		d.RedCount += deltaDust.RedCount
		d.OrangeCount += deltaDust.OrangeCount
		d.YellowCount += deltaDust.YellowCount
		d.GreenCount += deltaDust.GreenCount
		d.BlueCount += deltaDust.BlueCount
		d.IndigoCount += deltaDust.IndigoCount
		d.PurpleCount += deltaDust.PurpleCount
		d.WhiteCount += deltaDust.WhiteCount
		d.BlackCount += deltaDust.BlackCount
	})
	if err != nil {
		return nil, nil, Dust{}, Herb{}, err
	}
	dbHerb, err := s.s.UpdateHerb(userID, func(d *storage.Herb) {
		d.MelissaCount += deltaHerb.MelissaCount
		d.LavandaCount += deltaHerb.LavandaCount
	})
	if err != nil {
		return nil, nil, Dust{}, Herb{}, err
	}
	return mapDbDustToCurrentDust(dbDust), mapDbHerbToCurrentHerb(dbHerb), deltaDust, deltaHerb, nil
}

func mapDbDustToCurrentDust(dbDust *storage.Dust) *Dust {
	return &Dust{
		RedCount:    dbDust.RedCount,
		OrangeCount: dbDust.OrangeCount,
		YellowCount: dbDust.YellowCount,
		GreenCount:  dbDust.GreenCount,
		BlueCount:   dbDust.BlueCount,
		IndigoCount: dbDust.IndigoCount,
		PurpleCount: dbDust.PurpleCount,
		WhiteCount:  dbDust.WhiteCount,
		BlackCount:  dbDust.BlackCount,
	}
}
func mapDbHerbToCurrentHerb(dbHerb *storage.Herb) *Herb {
	return &Herb{
		LavandaCount: dbHerb.LavandaCount,
		MelissaCount: dbHerb.MelissaCount,
	}
}

func (dust Dust) TotalDust() int64 {
	return dust.RedCount + dust.OrangeCount + dust.YellowCount + dust.GreenCount + dust.BlueCount + dust.IndigoCount + dust.PurpleCount + dust.WhiteCount + dust.BlackCount
}

func (herb Herb) TotalHerb() int64 {
	return herb.LavandaCount + herb.MelissaCount
}

func (s *Service) GetLoot(userID int64) (*Dust, *Herb, error) {
	dbDust, err := s.s.GetDustByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	dbHerb, err := s.s.GetHerbByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	return mapDbDustToCurrentDust(&dbDust), mapDbHerbToCurrentHerb(&dbHerb), nil
}

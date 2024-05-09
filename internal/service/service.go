package service

import (
	"context"
	"math"
	"math/rand"
	"time"
	"unicode/utf8"

	"github.com/pechorka/adhd-reader/internal/storage"

	"github.com/pechorka/adhd-reader/pkg/chance"
	"github.com/pechorka/adhd-reader/pkg/randstring"
	"github.com/pechorka/adhd-reader/pkg/textspliter"
	"github.com/pechorka/adhd-reader/pkg/webscraper"
	"github.com/pkg/errors"
)

var ErrTextFinished = errors.New("text finished")
var ErrFirstChunk = errors.New("first chunk")
var ErrTextNotSelected = errors.New("text is not selected")
var ErrTextNotUTF8 = errors.New("text is not valid utf8")
var ErrInvalidToken = errors.New("invalid token")

const telegramMessageLengthLimit = 4096

type Chancer interface {
	Win(percent float64) bool
	PickWin(inputs ...chance.WinInput)
}

type Encryptor interface {
	EncryptString(plaintext string) (string, error)
	DecryptString(ciphertext string) (string, error)
}

type Service struct {
	s         *storage.Storage
	scrapper  *webscraper.WebScrapper
	chancer   Chancer
	encryptor Encryptor
	chunkSize int64
}

func NewService(
	s *storage.Storage,
	chunkSize int64,
	scrapper *webscraper.WebScrapper,
	encryptor Encryptor,
) *Service {
	return &Service{
		s:         s,
		chunkSize: chunkSize,
		chancer:   chance.Default,
		encryptor: encryptor,
		scrapper:  scrapper,
	}
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

func (s *Service) AddTextFromURL(userID int64, url string) (id string, name string, err error) {
	chunkSize, err := s.getChunkSize(userID)
	if err != nil {
		return "", "", err
	}
	name, text, err := s.scrapper.Scrape(context.Background(), url)
	if err != nil {
		return "", "", err
	}
	textChunks, err := s.processText(userID, name, text, chunkSize)
	if err != nil {
		return "", "", err
	}
	data := storage.NewText{
		Name:      name,
		Chunks:    textChunks,
		Text:      text,
		ChunkSize: chunkSize,
	}
	id, err = s.s.AddText(userID, data)
	return id, name, err
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
		pageSize = 40
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

func (s *Service) FullTexts(userID int64, after *time.Time) ([]storage.FullText, error) {
	return s.s.GetFullTexts(userID, after)
}

func calculateCompletionPercent(text storage.TextWithChunkInfo) int {
	if text.TotalChunks-1 <= 0 || text.CurrentChunk == storage.NotSelected {
		return 0
	}
	return int(float64(text.CurrentChunk) / float64(text.TotalChunks-1) * 100)
}

func (s *Service) QuickWin(userID int64) (storage.TextWithChunkInfo, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return storage.TextWithChunkInfo{}, err
	}

	minDelta := int64(math.MaxInt64)
	textI := -1
	for i, t := range texts {
		delta := t.TotalChunks - t.CurrentChunk
		if t.CurrentChunk == storage.NotSelected {
			delta = t.TotalChunks
		}
		if delta > 1 && delta < minDelta {
			textI = i
			minDelta = delta
		}
	}

	if textI == -1 {
		return storage.TextWithChunkInfo{}, errors.New("no texts found")
	}

	return texts[textI], nil
}

func (s *Service) RandomText(userID int64, atMostChunks int64) (storage.TextWithChunkInfo, error) {
	texts, err := s.s.GetTexts(userID)
	if err != nil {
		return storage.TextWithChunkInfo{}, err
	}
	if atMostChunks > 0 {
		texts = filterTextsByChunkCount(texts, atMostChunks)
	}

	unreadTexts := make([]storage.TextWithChunkInfo, 0, len(texts))
	for _, t := range texts {
		if !isTextFinished(t.CurrentChunk, t.TotalChunks) {
			unreadTexts = append(unreadTexts, t)
		}
	}

	if len(unreadTexts) == 0 {
		return storage.TextWithChunkInfo{}, errors.New("no unread texts")
	}

	return unreadTexts[rand.Intn(len(unreadTexts))], nil
}

func isTextFinished(curChunk, totalChunks int64) bool {
	return curChunk >= totalChunks-1
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

type SyncText struct {
	TextUUID     string
	CurrentChunk int64
	ModifiedAt   time.Time
	Deleted      bool
}

func (s *Service) SyncTexts(userID int64, texts []SyncText) ([]SyncText, error) {
	syncTextMap := make(map[string]SyncText, len(texts))
	for _, t := range texts {
		if t.Deleted {
			err := s.s.DeleteTextByUUID(userID, t.TextUUID)
			if err != nil {
				return nil, err
			}
			continue
		}
		syncTextMap[t.TextUUID] = t
	}
	var result []SyncText
	err := s.s.UpdateTexts(userID, func(texts *storage.UserTexts) error {
		for i := range texts.Texts {
			t := texts.Texts[i]
			syncText, ok := syncTextMap[t.UUID]
			if !ok {
				continue
			}
			delete(syncTextMap, t.UUID)
			if syncText.ModifiedAt.After(t.ModifiedAt) {
				t.CurrentChunk = syncText.CurrentChunk
				t.ModifiedAt = syncText.ModifiedAt
				texts.Texts[i] = t
				continue
			}
			// text on server is newer
			result = append(result, SyncText{
				TextUUID:     t.UUID,
				CurrentChunk: t.CurrentChunk,
				ModifiedAt:   t.ModifiedAt,
			})
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update texts")
	}
	// all texts that are left in syncTextMap are not found on server
	for _, t := range syncTextMap {
		result = append(result, SyncText{
			TextUUID: t.TextUUID,
			Deleted:  true,
		})
	}
	return result, err
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
		if isTextFinished(curChunk, totalChunks) {
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

func (c ChunkType) String() string {
	return string(c)
}

const (
	ChunkTypeFirst ChunkType = "first"
	ChunkTypeLast  ChunkType = "last"
	ChunkTypeOther ChunkType = "other"
)

func (s *Service) selectChunk(userID int64, selectChunk storage.SelectChunkFunc) (storage.Text, string, ChunkType, error) {
	var chunkType ChunkType = ChunkTypeOther
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

type Level struct {
	Level      int64
	Experience int64
}
type Stat struct {
	Free           int64
	Luck           int64
	Accuracy       int64
	Attention      int64
	TimeManagement int64
	Charizma       int64
}

type LootResult struct {
	DeltaDust *Dust
	TotalDust *Dust
	DeltaHerb *Herb
	TotalHerb *Herb
}

func (s *Service) LootOnNextChunk(userID int64) (*LootResult, error) {
	dbPlayerStats, err := s.s.GetStatByUserID(userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stats by user id")
	}
	var pointerdbPlayerStats *storage.Stat = &dbPlayerStats

	//@pechor, Это я побеждала указатели. Не удаляй пока, пожалуйста

	// var creature string = "shark"
	// var pointer *string = &creature

	// fmt.Println("creature =", creature)
	// fmt.Println("pointer =", pointer)
	// fmt.Println("*pointer =", *pointer)

	// Output
	// creature = shark
	// pointer = 0xc000010200
	// *pointer = shark

	playerStats := mapDbStatToServiceStat(pointerdbPlayerStats)
	deltaDust := s.findDustAtomicAction(playerStats.Attention, playerStats.Accuracy)
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
		return nil, errors.Wrap(err, "failed to update dust")
	}
	deltaHerb := s.findHerbAtomicAction(playerStats.Attention, playerStats.Accuracy)
	dbHerb, err := s.s.UpdateHerb(userID, func(d *storage.Herb) {
		d.MelissaCount += deltaHerb.MelissaCount
		d.LavandaCount += deltaHerb.LavandaCount
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update herb")
	}
	return &LootResult{
		DeltaDust: &deltaDust,
		TotalDust: mapDbDustToDust(dbDust),
		DeltaHerb: &deltaHerb,
		TotalHerb: mapDbHerbToHerb(dbHerb),
	}, nil
}

// currentLevel, deltaExp,levelUp?, err
func (s *Service) ExpOnNextChunk(userID int64) (*Level, int64, bool, error) {
	deltaExp := int64(2)
	//TODO determine by chunk size of TEXT not user
	userChunkSize, err := s.getChunkSize(userID)
	if err != nil {
		return nil, 0, false, err
	}
	deltaExp = calculateExperienceGainByChunkSize(userChunkSize)
	levelUp := false
	oldExperience, err := s.s.GetLevelByUserID(userID)
	if err != nil {
		return nil, 0, false, err
	}
	oldLevelNumber := DetectLevelByExperience(oldExperience.Experience)

	dbLevel, err := s.s.UpdateLevel(userID, func(d *storage.Level) {
		d.Experience += deltaExp
	})
	if err != nil {
		return nil, 0, false, err
	}

	newLevel := mapDbLevelToServiceLevel(dbLevel)
	if newLevel.Level > oldLevelNumber {
		levelUp = true
		if newLevel.Level >= 21 {
			_, err = s.s.UpdateStat(userID, func(d *storage.Stat) {
				d.Free++
			})
			if err != nil {
				return nil, 0, false, err
			}
		}
	}

	if newLevel.Level < 21 {
		currentStats := LevelUpStatDistribution(newLevel.Level)
		_, err = s.s.UpdateStat(userID, func(d *storage.Stat) {
			d.Accuracy = currentStats.Accuracy
			d.Attention = currentStats.Attention
			d.TimeManagement = currentStats.TimeManagement
			d.Charizma = currentStats.Charizma
			d.Luck = currentStats.Luck
		})
		if err != nil {
			return nil, 0, false, err
		}
	}
	//TODO: Display stats increase on level up

	return newLevel, deltaExp, levelUp, nil
}

func calculateExperienceGainByChunkSize(chunkSize int64) int64 {
	if chunkSize <= 0 {
		return int64(2)
	}
	exp := chunkSize / 250
	if exp < 1 {
		exp = 1
	}
	return exp
}

var levelThresholds = calculateLevelTresholds()

func DetectLevelByExperience(experience int64) int64 {

	newLevelNumber := int64(0)
	for _, threshold := range levelThresholds {
		if experience >= threshold {
			newLevelNumber += 1
		} else {
			break
		}
	}
	return newLevelNumber
}

func LevelUpStatDistribution(level int64) Stat {

	var accuracyByLevelIncrease = []int64{
		//01, 2, 3, 4, 5, 6, 7, 8, 9
		0, 1, 0, 1, 0, 1, 0, 1, 0, 0, //0-9
		0, 0, 0, 0, 0, 1, 0, 0, 0, 0, //10-19
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0} //20-29
	var attentionByLevelIncrease = []int64{
		0, 0, 1, 0, 1, 0, 1, 0, 0, 0, //0-9
		0, 1, 0, 0, 0, 0, 0, 0, 1, 0, //10-19
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0} //20-29
	var timeManagementByLevelIncrease = []int64{
		0, 0, 0, 0, 0, 0, 0, 0, 1, 0, //0-9
		1, 0, 0, 1, 0, 0, 1, 0, 0, 1, //10-19
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0} //20-29
	var charizmaByLevelIncrease = []int64{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //0-9
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, //10-19
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0} //20-29
	var luckByLevelIncrease = []int64{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 1, //0-9
		0, 0, 1, 0, 1, 0, 0, 1, 0, 0, //10-19
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0} //20-29

	var statsIncrease = [][]int64{
		accuracyByLevelIncrease,
		attentionByLevelIncrease,
		timeManagementByLevelIncrease,
		charizmaByLevelIncrease,
		luckByLevelIncrease,
	}
	var stats = Stat{}

	for i := 0; i < int(level)+1; i++ {
		stats.Accuracy += statsIncrease[0][i]
		stats.Attention += statsIncrease[1][i]
		stats.TimeManagement += statsIncrease[2][i]
		stats.Charizma += statsIncrease[3][i]
		stats.Luck += statsIncrease[4][i]
	}

	return stats
}

// TODO: Подумать над thresholds после 20 уровня. Они уже там не оч адекватно большие
func calculateLevelTresholds() []int64 {
	var levelThresholds = []int64{100}
	var levelRequirements = []int64{100}
	//filling levelRequirements with values. Each next level requires 1.1 times more than previous. Starting from 100 for Level 1
	for i := 1; i < 100; i++ {
		levelRequirements = append(levelRequirements, int64(float64(levelRequirements[i-1])*1.1))
	}
	for i := 1; i < len(levelRequirements); i++ {
		levelThresholds = append(levelThresholds, levelThresholds[i-1]+levelRequirements[i])
	}
	return levelThresholds
}

func (s *Service) findHerbAtomicAction(attention int64, accuracy int64) Herb {
	var deltaHerb Herb
	// BASE 1.9% chance to get Herb
	// 0.1% for each point in Attention
	var chanceToGetHerb float64 = 0.019 + 0.001*float64(attention)
	if !s.chancer.Win(chanceToGetHerb) {
		return deltaHerb
	}
	//Amount of herb depends on Accuracy. Each point of accuracy adds 33% chance to get one more herb
	s.chancer.PickWin(
		chance.WinInput{
			Percent: 0.7,
			Action: func() {
				deltaHerb.MelissaCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.33)
			},
		},
		chance.WinInput{
			Percent: 0.3,
			Action: func() {
				deltaHerb.LavandaCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.33)
			},
		},
	)
	return deltaHerb
}

func (s *Service) findDustAtomicAction(attention int64, accuracy int64) Dust {

	var deltaDust Dust
	// BASE 33% chance to get Dust
	// 0.1% for each point in Attention
	var chanceToGetDust float64 = 0.33 + 0.001*float64(attention)
	if !s.chancer.Win(chanceToGetDust) {
		return deltaDust
	}
	//Amount of dust depends on Accuracy. Each point of accuracy adds 50% chance to get one more dust
	s.chancer.PickWin(
		chance.WinInput{
			Percent: 0.25,
			Action: func() {
				deltaDust.RedCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.03,
			Action: func() {
				deltaDust.OrangeCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.25,
			Action: func() {
				deltaDust.YellowCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.142,
			Action: func() {
				deltaDust.GreenCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.13,
			Action: func() {
				deltaDust.BlueCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.13,
			Action: func() {
				deltaDust.IndigoCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.03,
			Action: func() {
				deltaDust.PurpleCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.019,
			Action: func() {
				deltaDust.WhiteCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
		chance.WinInput{
			Percent: 0.019,
			Action: func() {
				deltaDust.BlackCount += 1 + s.getLootAmountByAccuracyAndIncreaseRate(accuracy, 0.5)
			},
		},
	)
	return deltaDust
}

func (s *Service) getLootAmountByAccuracyAndIncreaseRate(accuracy int64, increaseRate float64) int64 {
	count := int64(0)
	for i := 0; i < int(accuracy); i++ {
		if s.chancer.Win(increaseRate) {
			count++
		}
	}
	return count
}

func mapDbDustToDust(dbDust *storage.Dust) *Dust {
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
func mapDbHerbToHerb(dbHerb *storage.Herb) *Herb {
	return &Herb{
		LavandaCount: dbHerb.LavandaCount,
		MelissaCount: dbHerb.MelissaCount,
	}
}

func mapDbLevelToServiceLevel(dbLevel *storage.Level) *Level {
	return &Level{
		Experience: dbLevel.Experience,
		Level:      DetectLevelByExperience(dbLevel.Experience),
	}
}

func mapDbStatToServiceStat(dbStat *storage.Stat) *Stat {
	return &Stat{
		Free:           dbStat.Free,
		Luck:           dbStat.Luck,
		Accuracy:       dbStat.Accuracy,
		Attention:      dbStat.Attention,
		TimeManagement: dbStat.TimeManagement,
		Charizma:       dbStat.Charizma,
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
	return mapDbDustToDust(&dbDust), mapDbHerbToHerb(&dbHerb), nil
}

func (s *Service) GetStatsAndLevel(userID int64) (*Stat, *Level, error) {
	dbStat, err := s.s.GetStatByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	dbLevel, err := s.s.GetLevelByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	return mapDbStatToServiceStat(&dbStat), mapDbLevelToServiceLevel(&dbLevel), nil
}

func (s *Service) GetAuthToken(userID int64) (string, error) {
	token, err := s.s.GetTokenByUserID(userID)
	switch err {
	case nil:
		return token, nil
	case storage.ErrNotFound:
	default:
		return "", err
	}
	return s.newToken(userID)
}

func (s *Service) ReIssueAuthToken(userID int64) (string, error) {
	err := s.s.DeleteAuthToken(userID)
	if err != nil && err != storage.ErrNotFound {
		return "", errors.Wrap(err, "failed to delete auth token")
	}
	return s.newToken(userID)
}

func (s *Service) ParseToken(token string) (int64, error) {
	rawToken, err := s.encryptor.DecryptString(token)
	if err != nil {
		return 0, ErrInvalidToken
	}
	userID, err := s.s.GetUserIDByAuthToken(rawToken)
	if err != nil {
		if err == storage.ErrNotFound {
			return 0, ErrInvalidToken
		}
		return 0, errors.Wrap(err, "failed to get user id by auth token")
	}
	return userID, nil
}

func (s *Service) newToken(userID int64) (string, error) {
	rawToken := randstring.Generate(32)
	token, err := s.encryptor.EncryptString(rawToken)
	if err != nil {
		return "", errors.Wrap(err, "failed to encrypt token")
	}
	err = s.s.SetAuthToken(userID, rawToken)
	if err != nil {
		return "", errors.Wrap(err, "failed to set auth token")
	}
	return token, nil
}

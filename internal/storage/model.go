package storage

import "time"

type UserTexts struct {
	Texts   []Text
	Current int // index of current text
}

type TextSource string

const (
	SourceText TextSource = "text"
	SourceFile TextSource = "file"
)

type Text struct {
	UUID         string
	Name         string
	Source       TextSource
	BucketName   []byte
	CurrentChunk int64
	CreatedAt    time.Time
	ModifiedAt   time.Time
}

type TextWithChunkInfo struct {
	UUID         string
	Name         string
	CurrentChunk int64
	TotalChunks  int64
}

type FullText struct {
	UUID         string
	Name         string
	CurrentChunk int64
	Chunks       []string
}

type NewText struct {
	Name      string
	Text      string
	Chunks    []string
	ChunkSize int64
}

type UserAnalytics struct {
	UserID         int64
	ChunkSize      int64
	TotalTextCount int64
	CurrentText    int
	Texts          []TextWithChunkInfo
}

type NewProcessedFile struct {
	Text      string
	Chunks    []string
	ChunkSize int64
	CheckSum  []byte
}

type ProcessedFile struct {
	UUID       string
	BucketName []byte
	ChunkSize  int64
	CheckSum   []byte
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

type Stat struct {
	Luck           int64
	Accuracy       int64
	Attention      int64
	TimeManagement int64
	Charizma       int64
}

type Level struct {
	Experience int64
}

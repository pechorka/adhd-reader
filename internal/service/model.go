package service

type ChunkType string

const (
	ChunkTypeFirst ChunkType = "first"
	ChunkTypeLast  ChunkType = "last"
)

type Text struct {
	UUID string
	Name string
}

type TextWithCompletion struct {
	UUID              string
	Name              string
	CompletionPercent int
}

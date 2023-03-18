package storage

type UserTexts struct {
	Texts   []Text
	Current int // index of current text
}

type Text struct {
	UUID       string
	Name       string
	BucketName []byte
}

type TextWithChunkInfo struct {
	UUID         string
	Name         string
	CurrentChunk int64
	TotalChunks  int64
}

type NewText struct {
	Name      string
	Text      string
	Chunks    []string
	ChunkSize int64
}
package storage

type UserTexts struct {
	Texts   []Text
	Current int // index of current text
}

func (ut UserTexts) getByUUID(uuid string) (Text, bool) {
	for _, t := range ut.Texts {
		if t.UUID == uuid {
			return t, true
		}
	}
	return Text{}, false
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

type FullTextInfo struct {
	UUID         string
	Name         string
	CurrentChunk int64
	Chunks       []string
	FullText     string
}

type NewText struct {
	Name      string
	Text      string
	Chunks    []string
	ChunkSize int64
}

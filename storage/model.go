package storage

type UserTexts struct {
	Texts   []Text
	Current int // index of current text
}

type Text struct {
	Name       string
	BucketName []byte
}

type NewText struct {
	Name   string
	Text   string
	Chunks []string
}

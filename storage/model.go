package storage

type UserTexts struct {
	Texts   []Text
	Current int // index of current text
}

type Text struct {
	Name     string
	Chunks   []string
	LastRead int // index of last read chunk
}

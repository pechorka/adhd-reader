package runeslice

import "unicode/utf8"

func NRunes(s string, n int) string {
	var i int
	for ; n > 0 && i < len(s); n-- {
		_, runeSize := utf8.DecodeRuneInString(s[i:])
		i += runeSize
	}
	return s[:i]
}

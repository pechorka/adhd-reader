package textspliter

import (
	"strings"
	"unicode/utf8"
)

func SplitText(text string, chunkSize int) []string {
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
		// backtracking to the nearest quote to check if we are in the middle of the quote
		for j = end; j > i && !isQuote(text[j]); j-- {
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

func isQuote(b byte) bool {
	return b == '"' || b == '\'' || b == '`'
}

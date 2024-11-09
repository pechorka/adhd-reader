package textspliter

import (
	"bytes"
	"strings"
)

func SplitText(text string, chunkSize int) []string {
	tokens := tokenize(text)
	buffer := bytes.NewBuffer(nil)
	buffer.Grow(chunkSize)
	chunks := make([]string, 0, len(text)/chunkSize)

	for i, token := range tokens {
		buffer.WriteString(token.Value)
		if buffer.Len() >= chunkSize { // got enough text, search for the nearest end of sentence
			if token.Type == EndSentence {
				if i+1 < len(tokens) && tokens[i+1].Type == EndSentence { // handle multiple punctuation marks, e.g. "!!!"
					continue
				}
				chunks = append(chunks, strings.TrimSpace(buffer.String()))
				buffer.Reset()
				continue
			}
		}
	}
	if buffer.Len() > 0 {
		chunks = append(chunks, buffer.String())
	}
	return chunks
}

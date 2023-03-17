package textspliter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_tokenize(t *testing.T) {
	// todo: handle link ending with punctuation mark
	// todo: handle ' in the middle of the word
	text := `This is a "sample" text, including some quotes and a link: https://www.example.com. Let's parse it!`
	tokens := tokenize(text)
	expectTockens := []Token{
		{Type: Word, Value: "This"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "is"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "a"},
		{Type: Space, Value: " "},
		{Type: BeginQuote, Value: `"`},
		{Type: Word, Value: "sample"},
		{Type: EndQuote, Value: `"`},
		{Type: Space, Value: " "},
		{Type: Word, Value: "text"},
		{Type: Punctuation, Value: ","},
		{Type: Space, Value: " "},
		{Type: Word, Value: "including"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "some"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "quotes"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "and"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "a"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "link"},
		{Type: Punctuation, Value: ":"},
		{Type: Space, Value: " "},
		{Type: Link, Value: "https://www.example.com"},
		{Type: EndSentence, Value: "."},
		{Type: Space, Value: " "},
		{Type: Word, Value: "Let's"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "parse"},
		{Type: Space, Value: " "},
		{Type: Word, Value: "it"},
		{Type: EndSentence, Value: "!"},
	}
	require.Len(t, tokens, len(expectTockens))
	for i := range expectTockens {
		require.Equal(t, expectTockens[i], tokens[i], "token %d: want %v, got %v", i, expectTockens[i], tokens[i])
	}
}

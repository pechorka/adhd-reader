package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_splitText(t *testing.T) {
	tests := []struct {
		name string
		text string
		size int
		want []string
	}{
		{name: "empty", text: "", size: 10, want: nil},
		{name: "single chunk", text: "text", size: 10, want: []string{"text"}},
		{
			name: "don't chunk sentence in the middle",
			text: "Sentence with a lot of words. It should be split into two chunks.",
			size: 4,
			want: []string{
				"Sentence with a lot of words.",
				"It should be split into two chunks.",
			},
		},
		{
			name: "cyrillic",
			text: "Текст. Не должен. Разбиться в середине rune.",
			size: 3,
			want: []string{
				"Текст.",
				"Не должен.",
				"Разбиться в середине rune.",
			},
		},
		{
			name: "multiple punctuation marks",
			text: "Sentence with a lot of punctuations?!? This text should be split into two chunks!!!!!",
			size: 4,
			want: []string{
				"Sentence with a lot of punctuations?!?",
				"This text should be split into two chunks!!!!!",
			},
		},
		{
			name: "skip until the end of the link",
			text: "Some words https://www.google.com/search?q=hello+world continue with the text. Next sentence.",
			size: 13,
			want: []string{
				"Some words https://www.google.com/search?q=hello+world continue with the text.",
				"Next sentence.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitText(tt.text, tt.size)
			for i, chunk := range got {
				require.Equal(t, tt.want[i], chunk)
			}
		})
	}
}

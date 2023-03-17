package textspliter

import (
	"bufio"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const (
	Word TokenType = iota
	Quote
	BeginQuote
	EndQuote
	EndSentence
	Punctuation
	Link
	Space
)

type Token struct {
	Type  TokenType
	Value string
}

func tokenize(text string) []Token {
	var tokens []Token

	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(tokenizer)

	inTheMiddleOfQuote := false
	for scanner.Scan() {
		token := scanner.Text()
		if token == "" {
			continue
		}
		tokenType := getTokenType(token)
		tokenType, inTheMiddleOfQuote = handleQuoteToken(tokenType, inTheMiddleOfQuote)
		tokens = append(tokens, Token{Type: tokenType, Value: token})
	}

	return tokens
}

var linkRegexp = regexp.MustCompile(`^(http://|https://|ftp://|www\.)`)
var quoteRegexp = regexp.MustCompile(`^("|'|«|„|“|` + "`" + `|»|”)`)

func tokenizer(data []byte, atEOF bool) (advance int, token []byte, err error) {
	nextPunctuationIsEndOfSentence := false
	for i := 0; i < len(data); i++ {
		if data[i] == ' ' {
			if i == 0 {
				return i + 1, data[i : i+1], nil
			}
			return i, data[:i], nil // return word first
		}
		if i == len(data)-1 && atEOF {
			return len(data), data[:], nil
		}
		if linkRegexp.Match(data[i:]) {
			j := i
			for ; j < len(data) && data[j] != ' '; j++ {
			}

			if ok, size := isPunctuationBefore(data, j); ok {
				j -= size // link ends with punctuation mark
			}
			return j, data[i:j], nil
		}
		if quoteRegexp.Match(data[i:]) {
			_, size := utf8.DecodeRune(data[i:])
			if i-1 > -1 && data[i-1] == ' ' { // begining of quote
				return i + size, data[i : i+size], nil
			}
			if i == 0 { // begining or end of quote
				return i + size, data[i : i+size], nil
			}
			// word with quote in the middle, for example: "Let's"
			if isInTheMiddleOfTheWordAt(data, i) {
				continue // continue until space
			}
			return i, data[:i], nil // return quoted word first
		}
		if ok, size := isPunctuationAt(data, i); ok {
			if i == 0 { // word before punctuation already parsed
				return i + size, data[i : i+size], nil
			}
			// word with punctuation in the middle, for example: "i.e."
			if isInTheMiddleOfTheWordAt(data, i) && isPunctuationAfterNextRune(data, i+size) {
				nextPunctuationIsEndOfSentence = true
				continue
			}
			if nextPunctuationIsEndOfSentence {
				nextPunctuationIsEndOfSentence = false
				continue
			}
			return i, data[:i], nil // return word first
		}
	}

	if atEOF {
		return 0, nil, bufio.ErrFinalToken
	}

	return 0, nil, nil
}

func isInTheMiddleOfTheWordAt(data []byte, i int) bool {
	if i+1 < len(data) {
		nextRune, _ := utf8.DecodeRune(data[i+1:])
		if !unicode.IsLetter(nextRune) {
			return false
		}
	}
	prevRune, _ := decodeRunBefore(data, i)
	return unicode.IsLetter(prevRune)
}

func decodeRunBefore(data []byte, i int) (prevRune rune, size int) {
	prevRune = utf8.RuneError
	j := i - 1
	for j > -1 && prevRune == utf8.RuneError {
		prevRune, size = utf8.DecodeLastRune(data[j:i])
		j--
	}
	return prevRune, size
}

func isPunctuationAfterNextRune(data []byte, i int) bool {
	_, size := utf8.DecodeRune(data[i:])
	ok, _ := isPunctuationAt(data, i+size)
	return ok
}

func isPunctuationAt(data []byte, i int) (bool, int) {
	r, size := utf8.DecodeRune(data[i:])
	return unicode.IsPunct(r), size
}

func isPunctuationBefore(data []byte, i int) (bool, int) {
	r, size := decodeRunBefore(data, i)
	return unicode.IsPunct(r), size
}

func getTokenType(token string) TokenType {
	switch token {
	case ".", "!", "?":
		return EndSentence
	case ",", ":", ";", "—", "-":
		return Punctuation
	case "\"", "'", "«", "»", "„", "`", "”", "“", "‘", "’":
		return Quote
	case " ":
		return Space
	default:
		if strings.HasPrefix(token, "http://") || strings.HasPrefix(token, "https://") ||
			strings.HasPrefix(token, "ftp://") || strings.HasPrefix(token, "www.") {
			return Link
		}
		return Word
	}
}

func handleQuoteToken(tokenType TokenType, inTheMiddleOfQuote bool) (TokenType, bool) {
	if tokenType != Quote {
		return tokenType, inTheMiddleOfQuote
	}
	if inTheMiddleOfQuote {
		return EndQuote, false
	}
	return BeginQuote, true
}

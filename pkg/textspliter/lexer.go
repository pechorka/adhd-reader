package textspliter

import (
	"bufio"
	"regexp"
	"strings"
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
var quoteRegexp = regexp.MustCompile(`^("|'|«|„|“|` + "`" + `|»|”|”)`)

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
			if j-1 > -1 && isPunctuation(data[j-1]) {
				j-- // link ends with punctuation mark
			}
			return j, data[i:j], nil
		}
		if quoteRegexp.Match(data[i:]) {
			if i-1 > -1 && data[i-1] == ' ' { // begining of quote
				return i + 1, data[i : i+1], nil
			}
			if i == 0 { // begining or end of quote
				return i + 1, data[i : i+1], nil
			}
			// word with quote in the middle, for example: "Let's"
			if isInTheMiddleOfTheWordAt(data, i) {
				continue // continue until space
			}
			return i, data[:i], nil // return quoted word first
		}
		if isPunctuation(data[i]) {
			if i == 0 { // word before punctuation already parsed
				return i + 1, data[i : i+1], nil
			}
			// word with punctuation in the middle, for example: "i.e."
			if isInTheMiddleOfTheWordAt(data, i) && isPunctuationAfterPuncuationAndNextRune(data, i) {
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
	return i-1 > -1 && data[i-1] != ' ' &&
		i+1 < len(data) && data[i+1] != ' ' && !isPunctuation(data[i+1])
}

func isPunctuationAfterPuncuationAndNextRune(data []byte, i int) bool {
	i++
	if i >= len(data) {
		return false
	}
	_, size := utf8.DecodeRune(data[i:])
	return i+size < len(data) && isPunctuation(data[i+size])
}

func isPunctuation(b byte) bool {
	switch b {
	case ',', '.', '!', '?', ':', ';':
		return true
	default:
		return false
	}
}

func getTokenType(token string) TokenType {
	switch token {
	case ".", "!", "?":
		return EndSentence
	case ",", ":", ";":
		return Punctuation
	case "\"", "'", "«", "„", "“", "`", "»":
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

package parser

import (
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenWord TokenType = iota
	TokenPipe
	TokenAnd
	TokenOr
	TokenSemi
	TokenAmp
	TokenRedirIn
	TokenRedirOut
	TokenRedirAppend
	TokenRedirErr
	TokenRedirErrApp
	TokenRedirErrToOut
	TokenRedirAll
	TokenEOF
)

type Token struct {
	Type  TokenType
	Value string
}

func Tokenize(input string) ([]Token, error) {
	var tokens []Token
	runes := []rune(input)
	n := len(runes)
	i := 0

	for i < n {

		for i < n && unicode.IsSpace(runes[i]) {
			i++
		}
		if i >= n {
			break
		}

		if i+3 < n && string(runes[i:i+4]) == "2>&1" {
			tokens = append(tokens, Token{Type: TokenRedirErrToOut, Value: "2>&1"})
			i += 4
			continue
		}
		if i+2 < n && string(runes[i:i+3]) == "2>>" {
			tokens = append(tokens, Token{Type: TokenRedirErrApp, Value: "2>>"})
			i += 3
			continue
		}
		if i+1 < n && string(runes[i:i+2]) == "2>" {
			tokens = append(tokens, Token{Type: TokenRedirErr, Value: "2>"})
			i += 2
			continue
		}
		if i+1 < n && string(runes[i:i+2]) == ">>" {
			tokens = append(tokens, Token{Type: TokenRedirAppend, Value: ">>"})
			i += 2
			continue
		}
		if i+1 < n && string(runes[i:i+2]) == "&>" {
			tokens = append(tokens, Token{Type: TokenRedirAll, Value: "&>"})
			i += 2
			continue
		}
		if i+1 < n && string(runes[i:i+2]) == "&&" {
			tokens = append(tokens, Token{Type: TokenAnd, Value: "&&"})
			i += 2
			continue
		}
		if i+1 < n && string(runes[i:i+2]) == "||" {
			tokens = append(tokens, Token{Type: TokenOr, Value: "||"})
			i += 2
			continue
		}

		switch runes[i] {
		case '|':
			tokens = append(tokens, Token{Type: TokenPipe, Value: "|"})
			i++
			continue
		case ';':
			tokens = append(tokens, Token{Type: TokenSemi, Value: ";"})
			i++
			continue
		case '&':
			tokens = append(tokens, Token{Type: TokenAmp, Value: "&"})
			i++
			continue
		case '<':
			tokens = append(tokens, Token{Type: TokenRedirIn, Value: "<"})
			i++
			continue
		case '>':
			tokens = append(tokens, Token{Type: TokenRedirOut, Value: ">"})
			i++
			continue
		}

		var word strings.Builder
		for i < n && !unicode.IsSpace(runes[i]) {

			if runes[i] == '|' || runes[i] == ';' || runes[i] == '&' || runes[i] == '<' || runes[i] == '>' {
				break
			}

			if runes[i] == '\\' {

				i++
				if i < n {
					word.WriteRune(runes[i])
					i++
				}
				continue
			}

			if runes[i] == '\'' {

				word.WriteRune('\'')
				i++
				for i < n && runes[i] != '\'' {
					word.WriteRune(runes[i])
					i++
				}
				if i < n && runes[i] == '\'' {
					word.WriteRune('\'')
					i++
				}
				continue
			}

			if runes[i] == '"' {

				word.WriteRune('"')
				i++
				for i < n && runes[i] != '"' {
					if runes[i] == '\\' && i+1 < n && (runes[i+1] == '"' || runes[i+1] == '\\' || runes[i+1] == '$') {
						word.WriteRune(runes[i])
						i++
						word.WriteRune(runes[i])
						i++
					} else {
						word.WriteRune(runes[i])
						i++
					}
				}
				if i < n && runes[i] == '"' {
					word.WriteRune('"')
					i++
				}
				continue
			}

			if runes[i] == '$' && i+1 < n && runes[i+1] == '(' {
				word.WriteString("$(")
				i += 2
				parenCount := 1
				for i < n && parenCount > 0 {
					if runes[i] == '(' {
						parenCount++
					} else if runes[i] == ')' {
						parenCount--
					}
					if parenCount > 0 {
						word.WriteRune(runes[i])
						i++
					}
				}
				if i < n && runes[i] == ')' {
					word.WriteRune(')')
					i++
				}
				continue
			}

			if runes[i] == '`' {
				word.WriteRune('`')
				i++
				for i < n && runes[i] != '`' {
					word.WriteRune(runes[i])
					i++
				}
				if i < n && runes[i] == '`' {
					word.WriteRune('`')
					i++
				}
				continue
			}

			word.WriteRune(runes[i])
			i++
		}

		if word.Len() > 0 {
			tokens = append(tokens, Token{Type: TokenWord, Value: word.String()})
		}
	}

	tokens = append(tokens, Token{Type: TokenEOF, Value: ""})
	return tokens, nil
}

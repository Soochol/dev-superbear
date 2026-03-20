package dsl

import (
	"fmt"
	"strings"
	"unicode"
)

// Tokenize takes a DSL input string and produces a slice of tokens.
func Tokenize(input string) ([]Token, error) {
	var tokens []Token
	runes := []rune(input)
	pos := 0
	line := 1
	col := 1

	for pos < len(runes) {
		ch := runes[pos]

		// Newline
		if ch == '\n' {
			tokens = append(tokens, Token{Type: TOKEN_NEWLINE, Value: "\n", Line: line, Column: col})
			pos++
			line++
			col = 1
			continue
		}

		// Skip whitespace (except newlines)
		if ch == ' ' || ch == '\t' || ch == '\r' {
			pos++
			col++
			continue
		}

		// String literal
		if ch == '"' || ch == '\'' {
			quote := ch
			startCol := col
			pos++
			col++
			var sb strings.Builder
			for pos < len(runes) && runes[pos] != quote {
				if runes[pos] == '\\' && pos+1 < len(runes) {
					pos++
					col++
					switch runes[pos] {
					case 'n':
						sb.WriteRune('\n')
					case 't':
						sb.WriteRune('\t')
					case '\\':
						sb.WriteRune('\\')
					default:
						sb.WriteRune(runes[pos])
					}
				} else {
					sb.WriteRune(runes[pos])
				}
				pos++
				col++
			}
			if pos >= len(runes) {
				return nil, fmt.Errorf("unterminated string at line %d, column %d", line, startCol)
			}
			pos++ // closing quote
			col++
			tokens = append(tokens, Token{Type: TOKEN_STRING, Value: sb.String(), Line: line, Column: startCol})
			continue
		}

		// Number (integer or float)
		if unicode.IsDigit(ch) {
			startCol := col
			var sb strings.Builder
			for pos < len(runes) && unicode.IsDigit(runes[pos]) {
				sb.WriteRune(runes[pos])
				pos++
				col++
			}
			if pos < len(runes) && runes[pos] == '.' {
				sb.WriteRune('.')
				pos++
				col++
				for pos < len(runes) && unicode.IsDigit(runes[pos]) {
					sb.WriteRune(runes[pos])
					pos++
					col++
				}
			}
			tokens = append(tokens, Token{Type: TOKEN_NUMBER, Value: sb.String(), Line: line, Column: startCol})
			continue
		}

		// Identifier or keyword
		if unicode.IsLetter(ch) || ch == '_' {
			startCol := col
			var sb strings.Builder
			for pos < len(runes) && (unicode.IsLetter(runes[pos]) || unicode.IsDigit(runes[pos]) || runes[pos] == '_') {
				sb.WriteRune(runes[pos])
				pos++
				col++
			}
			word := sb.String()
			if tt, ok := keywords[word]; ok {
				tokens = append(tokens, Token{Type: tt, Value: word, Line: line, Column: startCol})
			} else {
				tokens = append(tokens, Token{Type: TOKEN_IDENTIFIER, Value: word, Line: line, Column: startCol})
			}
			continue
		}

		// Operators and punctuation
		startCol := col
		switch ch {
		case '+':
			tokens = append(tokens, Token{Type: TOKEN_PLUS, Value: "+", Line: line, Column: startCol})
			pos++
			col++
		case '-':
			tokens = append(tokens, Token{Type: TOKEN_MINUS, Value: "-", Line: line, Column: startCol})
			pos++
			col++
		case '*':
			tokens = append(tokens, Token{Type: TOKEN_STAR, Value: "*", Line: line, Column: startCol})
			pos++
			col++
		case '/':
			tokens = append(tokens, Token{Type: TOKEN_SLASH, Value: "/", Line: line, Column: startCol})
			pos++
			col++
		case '(':
			tokens = append(tokens, Token{Type: TOKEN_LPAREN, Value: "(", Line: line, Column: startCol})
			pos++
			col++
		case ')':
			tokens = append(tokens, Token{Type: TOKEN_RPAREN, Value: ")", Line: line, Column: startCol})
			pos++
			col++
		case ',':
			tokens = append(tokens, Token{Type: TOKEN_COMMA, Value: ",", Line: line, Column: startCol})
			pos++
			col++
		case '.':
			tokens = append(tokens, Token{Type: TOKEN_DOT, Value: ".", Line: line, Column: startCol})
			pos++
			col++
		case '=':
			if pos+1 < len(runes) && runes[pos+1] == '=' {
				tokens = append(tokens, Token{Type: TOKEN_EQ, Value: "==", Line: line, Column: startCol})
				pos += 2
				col += 2
			} else {
				tokens = append(tokens, Token{Type: TOKEN_ASSIGN, Value: "=", Line: line, Column: startCol})
				pos++
				col++
			}
		case '!':
			if pos+1 < len(runes) && runes[pos+1] == '=' {
				tokens = append(tokens, Token{Type: TOKEN_NEQ, Value: "!=", Line: line, Column: startCol})
				pos += 2
				col += 2
			} else {
				return nil, fmt.Errorf("unexpected character '!' at line %d, column %d", line, col)
			}
		case '<':
			if pos+1 < len(runes) && runes[pos+1] == '=' {
				tokens = append(tokens, Token{Type: TOKEN_LTE, Value: "<=", Line: line, Column: startCol})
				pos += 2
				col += 2
			} else {
				tokens = append(tokens, Token{Type: TOKEN_LT, Value: "<", Line: line, Column: startCol})
				pos++
				col++
			}
		case '>':
			if pos+1 < len(runes) && runes[pos+1] == '=' {
				tokens = append(tokens, Token{Type: TOKEN_GTE, Value: ">=", Line: line, Column: startCol})
				pos += 2
				col += 2
			} else {
				tokens = append(tokens, Token{Type: TOKEN_GT, Value: ">", Line: line, Column: startCol})
				pos++
				col++
			}
		default:
			return nil, fmt.Errorf("unexpected character '%c' at line %d, column %d", ch, line, col)
		}
	}

	tokens = append(tokens, Token{Type: TOKEN_EOF, Value: "", Line: line, Column: col})
	return tokens, nil
}

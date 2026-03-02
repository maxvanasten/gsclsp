package analysis

import (
	"bytes"

	l "github.com/maxvanasten/gscp/lexer"
)

type TokenType int

const (
	VARIABLE TokenType = iota
	KEYWORD
	STRING
	NUMBER
	FUNCTION
	PROPERTY
)

func GenerateSemanticTokens(tokens []l.Token) []int {
	semanticTokens := []int{}

	prevLine := 0
	prevChar := 0
	emit := func(line, col, length int, tokenType TokenType) {
		deltaLine := line - prevLine
		deltaStart := col
		if deltaLine == 0 {
			deltaStart = col - prevChar
		}
		semanticTokens = append(semanticTokens, deltaLine, deltaStart, length, int(tokenType), 0)
		prevLine = line
		prevChar = col
	}

	for i, t := range tokens {
		line := t.Line - 1
		col := t.Col - 1

		switch t.Type {
		case l.SYMBOL:
			if bytes.Contains([]byte(t.Content), []byte{'\\'}) {
				emit(line, col, len(t.Content), STRING)
			} else if bytes.Contains([]byte(t.Content), []byte{'.'}) {
				// TODO: Handle multiple . in a variable like player.weapon.name
				object, prop, _ := bytes.Cut([]byte(t.Content), []byte{'.'})

				emit(line, col, len(object), VARIABLE)
				if len(prop) > 0 {
					emit(line, col+len(object)+1, len(prop), PROPERTY)
				}
			} else {
				isFunctionCall := false
				// Check if next token is open_paren
				if i+1 < len(tokens) {
					if tokens[i+1].Type == l.OPEN_PAREN {
						emit(line, col, len(t.Content), FUNCTION)
						isFunctionCall = true
					}
				}
				if !isFunctionCall {
					// Check if keyword
					switch t.Content {
					case "thread", "wait", "#include", "case", "break", "default", "return", "true", "false", "if", "else", "for":
						emit(line, col, len(t.Content), KEYWORD)
					default:
						emit(line, col, len(t.Content), VARIABLE)
					}
				}
			}
		case l.STRING:
			emit(line, col, len(t.Content)+2, STRING)
		}
	}

	return semanticTokens
}

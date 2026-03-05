package analysis

import (
	"strings"

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
	COMMENT
)

var keywordTokens = map[string]struct{}{
	"thread": {}, "wait": {}, "#include": {}, "case": {}, "break": {}, "default": {},
	"return": {}, "true": {}, "false": {}, "if": {}, "else": {}, "for": {}, "foreach": {},
	"while": {}, "do": {}, "continue": {}, "waittill": {}, "waittillmatch": {},
	"waittillframeend": {}, "endon": {}, "self": {}, "level": {}, "switch": {}, "in": {},
	"notify": {}, "breakpoint": {},
}

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
		// SYMBOL can be either variable, function name, parameter, file_path
		// This really should be handled by the lexer already.
		case l.SYMBOL:
			if isIncludePathToken(tokens, i) || strings.ContainsAny(t.Content, "\\/") {
				// TODO: find better token type for file path
				emit(line, col, len(t.Content), STRING)
				break
			}

			if strings.Contains(t.Content, ".") {
				// TODO: Handle multiple . in a variable like player.weapon.name
				object, prop, _ := strings.Cut(t.Content, ".")

				emit(line, col, len(object), VARIABLE)
				if prop != "" {
					emit(line, col+len(object)+1, len(prop), PROPERTY)
				}
				break
			}

			// Check if keyword
			if _, ok := keywordTokens[t.Content]; ok {
				emit(line, col, len(t.Content), KEYWORD)
				break
			}
			// Check if next token is open_paren
			if i+1 < len(tokens) && tokens[i+1].Type == l.OPEN_PAREN {
				emit(line, col, len(t.Content), FUNCTION)
				break
			}
			emit(line, col, len(t.Content), VARIABLE)
		case l.STRING:
			emit(line, col, len(t.Content)+2, STRING)
		case l.NUMBER:
			emit(line, col, len(t.Content), NUMBER)
		case l.LINE_COMMENT:
			emit(line, col, len(t.Content), COMMENT)
		case l.BLOCK_COMMENT:
			segments := strings.Split(t.Content, "\n")
			for segmentIndex, segment := range segments {
				if segment == "" {
					continue
				}
				segmentLine := line + segmentIndex
				segmentCol := 0
				if segmentIndex == 0 {
					segmentCol = col
				}
				emit(segmentLine, segmentCol, len(segment), COMMENT)
			}
		}
	}

	return semanticTokens
}

func isIncludePathToken(tokens []l.Token, index int) bool {
	if index <= 0 || index >= len(tokens) {
		return false
	}
	prev := tokens[index-1]
	current := tokens[index]
	return prev.Type == l.SYMBOL && prev.Content == "#include" && prev.Line == current.Line
}

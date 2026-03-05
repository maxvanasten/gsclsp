package analysis

import "testing"

type decodedSemanticToken struct {
	Line   int
	Col    int
	Length int
	Type   TokenType
}

func decodeSemanticTokens(data []int) []decodedSemanticToken {
	decoded := []decodedSemanticToken{}
	prevLine := 0
	prevChar := 0
	for i := 0; i+4 < len(data); i += 5 {
		deltaLine := data[i]
		deltaStart := data[i+1]
		length := data[i+2]
		tokenType := TokenType(data[i+3])

		line := prevLine + deltaLine
		col := deltaStart
		if deltaLine == 0 {
			col = prevChar + deltaStart
		}

		decoded = append(decoded, decodedSemanticToken{
			Line:   line,
			Col:    col,
			Length: length,
			Type:   tokenType,
		})
		prevLine = line
		prevChar = col
	}

	return decoded
}

func TestSemanticTokensIncludePathWithoutSeparators(t *testing.T) {
	requireGscp(t)
	input := "#include file;\n"
	parseResult := Parse(input)
	semanticTokens := GenerateSemanticTokens(parseResult.Tokens)
	decoded := decodeSemanticTokens(semanticTokens)

	fileTokenIndex := -1
	for i := range parseResult.Tokens {
		if parseResult.Tokens[i].Content == "file" {
			fileTokenIndex = i
			break
		}
	}
	if fileTokenIndex == -1 {
		t.Fatalf("missing lexer token for include path")
	}

	fileToken := parseResult.Tokens[fileTokenIndex]
	fileLine := fileToken.Line - 1
	fileCol := fileToken.Col - 1
	fileLength := len(fileToken.Content)
	for _, token := range decoded {
		if token.Line == fileLine && token.Col == fileCol && token.Length == fileLength {
			if token.Type != STRING {
				t.Fatalf("include path token type mismatch: got %v want %v", token.Type, STRING)
			}
			return
		}
	}

	t.Fatalf("missing semantic token for include path")
}

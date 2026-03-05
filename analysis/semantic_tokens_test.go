package analysis

import (
	"strings"
	"testing"

	l "github.com/maxvanasten/gscp/lexer"
)

type decodedSemanticToken struct {
	Line   int
	Col    int
	Length int
	Type   TokenType
}

func TestSemanticTokensLineComment(t *testing.T) {
	input := "main(){\n// keep\n}"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	data := GenerateSemanticTokens(tokens)
	decoded := decodeSemanticTokens(data)

	var commentToken l.Token
	foundLexerComment := false
	for _, tok := range tokens {
		if tok.Type == l.LINE_COMMENT {
			commentToken = tok
			foundLexerComment = true
			break
		}
	}
	if !foundLexerComment {
		t.Fatal("missing lexer line comment token")
	}

	line := commentToken.Line - 1
	col := commentToken.Col - 1
	length := len(commentToken.Content)
	for _, token := range decoded {
		if token.Line == line && token.Col == col && token.Length == length {
			if token.Type != COMMENT {
				t.Fatalf("expected comment semantic token, got %v", token.Type)
			}
			return
		}
	}

	t.Fatal("missing semantic token for line comment")
}

func TestSemanticTokensMultilineBlockComment(t *testing.T) {
	input := "main(){\n/# keep\nblock #/\n}"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	data := GenerateSemanticTokens(tokens)
	decoded := decodeSemanticTokens(data)

	var blockComment l.Token
	foundBlock := false
	for _, tok := range tokens {
		if tok.Type == l.BLOCK_COMMENT {
			blockComment = tok
			foundBlock = true
			break
		}
	}
	if !foundBlock {
		t.Fatal("missing lexer block comment token")
	}

	segments := strings.Split(blockComment.Content, "\n")
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		line := blockComment.Line - 1 + i
		col := 0
		if i == 0 {
			col = blockComment.Col - 1
		}
		matched := false
		for _, token := range decoded {
			if token.Line == line && token.Col == col && token.Length == len(segment) {
				if token.Type != COMMENT {
					t.Fatalf("expected comment semantic token, got %v", token.Type)
				}
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("missing semantic token for comment segment %q", segment)
		}
	}
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
	parseResult, err := Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
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

func TestSemanticTokensKeywordOverFunction(t *testing.T) {
	input := "if(true){switch(true){for(i=0;i<1;i++){}}}"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	data := GenerateSemanticTokens(tokens)
	decoded := decodeSemanticTokens(data)

	seenKeyword := map[string]bool{"if": false, "switch": false, "for": false}
	for _, token := range tokens {
		if token.Type != l.SYMBOL {
			continue
		}
		if _, ok := seenKeyword[token.Content]; !ok {
			continue
		}
		line := token.Line - 1
		col := token.Col - 1
		length := len(token.Content)
		matched := false
		for _, decodedToken := range decoded {
			if decodedToken.Line == line && decodedToken.Col == col && decodedToken.Length == length {
				matched = true
				if decodedToken.Type != KEYWORD {
					t.Fatalf("expected keyword token for %s, got %v", token.Content, decodedToken.Type)
				}
				break
			}
		}
		if !matched {
			t.Fatalf("missing semantic token for %s", token.Content)
		}
		seenKeyword[token.Content] = true
	}
	for name, ok := range seenKeyword {
		if !ok {
			t.Fatalf("missing keyword token for %s", name)
		}
	}
}

func TestSemanticTokensBO2Keywords(t *testing.T) {
	input := "foreach(i in arr){} while(true){} do{}while(false); continue; waittillmatch(\"evt\"); waittillframeend; breakpoint;"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	data := GenerateSemanticTokens(tokens)
	decoded := decodeSemanticTokens(data)

	seenKeyword := map[string]bool{
		"foreach":          false,
		"in":               false,
		"while":            false,
		"do":               false,
		"continue":         false,
		"waittillmatch":    false,
		"waittillframeend": false,
		"breakpoint":       false,
	}
	for _, token := range tokens {
		if token.Type != l.SYMBOL {
			continue
		}
		if _, ok := seenKeyword[token.Content]; !ok {
			continue
		}
		line := token.Line - 1
		col := token.Col - 1
		length := len(token.Content)
		matched := false
		for _, decodedToken := range decoded {
			if decodedToken.Line == line && decodedToken.Col == col && decodedToken.Length == length {
				matched = true
				if decodedToken.Type != KEYWORD {
					t.Fatalf("expected keyword token for %s, got %v", token.Content, decodedToken.Type)
				}
				break
			}
		}
		if !matched {
			t.Fatalf("missing semantic token for %s", token.Content)
		}
		seenKeyword[token.Content] = true
	}
	for name, ok := range seenKeyword {
		if !ok {
			t.Fatalf("missing keyword token for %s", name)
		}
	}
}

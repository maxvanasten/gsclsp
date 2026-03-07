package analysis

import (
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

type InlayHintResolution struct {
	Signature   FunctionSignature
	OriginLabel string
	ShowOrigin  bool
}

type InlayHintResolver func(name string) (InlayHintResolution, bool)

const maxTokenCountForOpenCallHints = 4000

func GenerateInlayHints(signatures []FunctionSignature, nodes []p.Node, tokens []l.Token, resolve InlayHintResolver) []lsp.InlayHint {
	hints := []lsp.InlayHint{}
	if resolve == nil {
		resolve = func(name string) (InlayHintResolution, bool) {
			sig, ok := findSignatureByName(signatures, name)
			if !ok {
				return InlayHintResolution{}, false
			}
			return InlayHintResolution{Signature: sig}, true
		}
	}

	allowOpenCallHints := len(tokens) <= maxTokenCountForOpenCallHints
	tokenIndex := indexTokensByLine(tokens)

	var walk func([]p.Node)
	walk = func(nodes []p.Node) {
		for _, n := range nodes {
			if n.Type != "function_call" {
				if len(n.Children) > 0 {
					walk(n.Children)
				}
				continue
			}

			callName := functionCallName(n)
			resolved, ok := resolve(callName)
			if !ok || len(resolved.Signature.Arguments) == 0 {
				continue
			}
			if n.Line <= 0 || n.Col <= 0 {
				continue
			}
			labels := resolved.Signature.Arguments
			anchorLine := n.Line - 1
			if anchorLine < 0 {
				anchorLine = 0
			}
			lineTokens := tokenIndex[n.Line]
			callCol := functionCallLabelCol(n, lineTokens)
			anchorCol := callCol
			if n.Data.FunctionName != "" {
				anchorCol = callCol + len(n.Data.FunctionName) + 1
			}
			if anchorCol < 0 {
				anchorCol = 0
			}
			if resolved.ShowOrigin && resolved.OriginLabel != "" {
				hints = append(hints, lsp.InlayHint{
					Position: lsp.Position{
						Line:      anchorLine,
						Character: callCol,
					},
					Label: resolved.OriginLabel,
				})
			}

			if len(n.Children) > 0 {
				for i, a := range n.Children {
					if i >= len(labels) {
						break
					}
					label := labels[i] + ": "
					col := a.Col - 1
					if col <= 0 {
						col = anchorCol
					}
					hints = append(hints, lsp.InlayHint{
						Position: lsp.Position{
							Line:      anchorLine,
							Character: col,
						},
						Label: label,
					})
				}
				continue
			}

			if !allowOpenCallHints {
				continue
			}

			if !callClosedOnLine(lineTokens, callName) {
				paramIndex, stubCol, ok := openCallParamAnchor(lineTokens, callName)
				if !ok || paramIndex >= len(labels) {
					continue
				}
				label := labels[paramIndex] + ": "
				if stubCol < 0 {
					stubCol = anchorCol
				}
				hints = append(hints, lsp.InlayHint{
					Position: lsp.Position{
						Line:      anchorLine,
						Character: stubCol,
					},
					Label: label,
				})
			}
		}
	}

	walk(nodes)

	return hints
}

func callClosedOnLine(lineTokens []l.Token, functionName string) bool {
	for i, t := range lineTokens {
		if t.Type != l.SYMBOL || !tokenMatchesFunction(t.Content, functionName) {
			continue
		}
		seenOpen := false
		depthParen := 0
		for j := i + 1; j < len(lineTokens); j++ {
			if lineTokens[j].Type == l.NEWLINE || lineTokens[j].Type == l.TERMINATOR {
				break
			}
			if lineTokens[j].Type == l.OPEN_PAREN {
				if depthParen == 0 {
					seenOpen = true
				}
				depthParen++
				continue
			}
			if lineTokens[j].Type == l.CLOSE_PAREN {
				if depthParen > 0 {
					depthParen--
					if depthParen == 0 {
						return seenOpen
					}
				}
			}
		}
	}
	return false
}

func openCallParamAnchor(lineTokens []l.Token, functionName string) (int, int, bool) {
	for i, t := range lineTokens {
		if t.Type != l.SYMBOL || !tokenMatchesFunction(t.Content, functionName) {
			continue
		}
		seenOpen := false
		depthParen := 0
		depthBracket := 0
		depthCurly := 0
		commaCount := 0
		currentCol := 0
		for j := i + 1; j < len(lineTokens); j++ {
			if lineTokens[j].Type == l.NEWLINE || lineTokens[j].Type == l.TERMINATOR {
				break
			}
			if lineTokens[j].Type == l.OPEN_PAREN {
				if depthParen == 0 {
					seenOpen = true
					currentCol = lineTokens[j].EndCol + 1
				}
				depthParen++
				continue
			}
			if !seenOpen {
				continue
			}
			switch lineTokens[j].Type {
			case l.CLOSE_PAREN:
				if depthParen > 0 {
					depthParen--
				}
			case l.OPEN_BRACKET:
				depthBracket++
			case l.CLOSE_BRACKET:
				if depthBracket > 0 {
					depthBracket--
				}
			case l.OPEN_CURLY:
				depthCurly++
			case l.CLOSE_CURLY:
				if depthCurly > 0 {
					depthCurly--
				}
			case l.COMMA:
				if depthParen == 1 && depthBracket == 0 && depthCurly == 0 {
					commaCount++
					currentCol = lineTokens[j].EndCol + 1
				}
			}
		}
		if seenOpen {
			return commaCount, currentCol - 1, true
		}
	}
	return 0, 0, false
}

func tokenMatchesFunction(tokenContent, functionName string) bool {
	if tokenContent == functionName {
		return true
	}
	if strings.HasSuffix(tokenContent, "::"+functionName) {
		return true
	}
	if strings.HasSuffix(functionName, "::"+tokenContent) {
		return true
	}
	if _, funcName, ok := splitQualifiedName(tokenContent); ok {
		return funcName == functionName
	}
	if _, funcName, ok := splitQualifiedName(functionName); ok {
		return tokenContent == funcName
	}
	return false
}

func functionCallName(n p.Node) string {
	if n.Data.Path != "" {
		return n.Data.Path + "::" + n.Data.FunctionName
	}
	return n.Data.FunctionName
}

func functionCallLabelCol(n p.Node, lineTokens []l.Token) int {
	if col, ok := functionCallLabelColFromTokens(n, lineTokens); ok {
		return col
	}

	col := n.Col - 1
	if col < 0 {
		col = 0
	}
	if n.Data.Method == "" {
		if n.Data.Thread {
			col += len("thread ")
		}
		return col
	}
	col += len(n.Data.Method) + 1
	if n.Data.Thread {
		col += len("thread ")
	}
	if col < 0 {
		return 0
	}
	return col
}

func functionCallLabelColFromTokens(n p.Node, lineTokens []l.Token) (int, bool) {
	if len(lineTokens) == 0 {
		return 0, false
	}

	callName := functionCallName(n)
	if callName == "" {
		return 0, false
	}

	startCol := n.Col - 1
	if startCol < 0 {
		startCol = 0
	}

	closestBeforeCol := -1
	closestBeforeDistance := 0
	for i, tok := range lineTokens {
		if tok.Type != l.SYMBOL || !tokenMatchesFunction(tok.Content, callName) {
			continue
		}
		if !symbolStartsFunctionCall(lineTokens, i) {
			continue
		}

		candidateCol := tokenFunctionNameCol(tok.Content, n.Data.FunctionName, tok.Col-1)
		if candidateCol >= startCol {
			return candidateCol, true
		}

		distance := startCol - candidateCol
		if closestBeforeCol < 0 || distance < closestBeforeDistance {
			closestBeforeCol = candidateCol
			closestBeforeDistance = distance
		}
	}

	if closestBeforeCol >= 0 {
		return closestBeforeCol, true
	}

	return 0, false
}

func symbolStartsFunctionCall(lineTokens []l.Token, index int) bool {
	for i := index + 1; i < len(lineTokens); i++ {
		switch lineTokens[i].Type {
		case l.OPEN_PAREN:
			return true
		case l.LINE_COMMENT, l.BLOCK_COMMENT:
			continue
		case l.NEWLINE, l.TERMINATOR:
			return false
		default:
			return false
		}
	}

	return false
}

func tokenFunctionNameCol(tokenContent, functionName string, tokenCol int) int {
	if tokenCol < 0 {
		tokenCol = 0
	}
	if functionName == "" {
		return tokenCol
	}
	if tokenContent == functionName {
		return tokenCol
	}
	if strings.HasSuffix(tokenContent, "::"+functionName) {
		return tokenCol + len(tokenContent) - len(functionName)
	}
	if _, tokenFunctionName, ok := splitQualifiedName(tokenContent); ok && tokenFunctionName == functionName {
		return tokenCol + len(tokenContent) - len(functionName)
	}
	return tokenCol
}

func indexTokensByLine(tokens []l.Token) map[int][]l.Token {
	if len(tokens) == 0 {
		return map[int][]l.Token{}
	}
	indexed := make(map[int][]l.Token)
	for _, t := range tokens {
		indexed[t.Line] = append(indexed[t.Line], t)
	}
	return indexed
}

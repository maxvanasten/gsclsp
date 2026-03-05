package analysis

import (
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

func GenerateInlayHints(signatures []FunctionSignature, nodes []p.Node, tokens []l.Token, resolve SignatureResolver) []lsp.InlayHint {
	hints := []lsp.InlayHint{}
	if resolve == nil {
		resolve = func(name string) (FunctionSignature, bool) {
			return findSignatureByName(signatures, name)
		}
	}

	for _, n := range nodes {
		if n.Type == "function_call" {
			if len(n.Children) > 0 {
				lookupName := functionCallLookupName(n)
				sig, ok := resolve(lookupName)
				if ok && len(sig.Arguments) > 0 {
					labels := sig.Arguments
					if n.Line <= 0 || n.Col <= 0 {
						continue
					}
					anchorLine := n.Line - 1
					if anchorLine < 0 {
						anchorLine = 0
					}
					anchorCol := n.Col - 1
					if n.Data.FunctionName != "" {
						anchorCol = n.Col - 1 + len(n.Data.FunctionName) + 1
					}
					if anchorCol < 0 {
						anchorCol = 0
					}
					callName := functionCallTokenName(n)
					if !callClosedOnLine(n, tokens, callName) {
						paramIndex, stubCol, ok := openCallParamAnchor(n, tokens, callName)
						if !ok || paramIndex >= len(labels) {
							continue
						}
						label := strings.Builder{}
						label.WriteString(labels[paramIndex])
						label.WriteString(": ")
						if stubCol < 0 {
							stubCol = anchorCol
						}
						hints = append(hints, lsp.InlayHint{
							Position: lsp.Position{
								Line:      anchorLine,
								Character: stubCol,
							},
							Label: label.String(),
						})
						continue
					}
					for i, a := range n.Children {
						if i < len(labels) {
							label := strings.Builder{}
							label.WriteString(labels[i])
							label.WriteString(": ")
							line := anchorLine
							col := a.Col - 1
							if col <= 0 {
								col = anchorCol
							}
							hints = append(hints, lsp.InlayHint{
								Position: lsp.Position{
									Line:      line,
									Character: col,
								},
								Label: label.String(),
							})
						}
					}
				}
			}
		} else {
			if len(n.Children) > 0 {
				hints = append(hints, GenerateInlayHints(signatures, n.Children, tokens, resolve)...)
			}
		}
	}

	return hints
}

func callClosedOnLine(n p.Node, tokens []l.Token, functionName string) bool {
	if n.Line <= 0 || n.Data.FunctionName == "" {
		return false
	}
	for i, t := range tokens {
		if t.Type != l.SYMBOL || !tokenMatchesFunction(t.Content, functionName) || t.Line != n.Line {
			continue
		}
		seenOpen := false
		depthParen := 0
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].Type == l.NEWLINE || tokens[j].Type == l.TERMINATOR {
				break
			}
			if tokens[j].Type == l.OPEN_PAREN {
				if depthParen == 0 {
					seenOpen = true
				}
				depthParen++
				continue
			}
			if tokens[j].Type == l.CLOSE_PAREN {
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

func openCallParamAnchor(n p.Node, tokens []l.Token, functionName string) (int, int, bool) {
	if n.Line <= 0 || n.Data.FunctionName == "" {
		return 0, 0, false
	}
	for i, t := range tokens {
		if t.Type != l.SYMBOL || !tokenMatchesFunction(t.Content, functionName) || t.Line != n.Line {
			continue
		}
		seenOpen := false
		depthParen := 0
		depthBracket := 0
		depthCurly := 0
		commaCount := 0
		currentCol := 0
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].Type == l.NEWLINE || tokens[j].Type == l.TERMINATOR {
				break
			}
			if tokens[j].Type == l.OPEN_PAREN {
				if depthParen == 0 {
					seenOpen = true
					currentCol = tokens[j].EndCol + 1
				}
				depthParen++
				continue
			}
			if !seenOpen {
				continue
			}
			switch tokens[j].Type {
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
					currentCol = tokens[j].EndCol + 1
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

func functionCallLookupName(n p.Node) string {
	if n.Data.Path != "" {
		return n.Data.Path + "::" + n.Data.FunctionName
	}
	return n.Data.FunctionName
}

func functionCallTokenName(n p.Node) string {
	if n.Data.Path != "" {
		return n.Data.Path + "::" + n.Data.FunctionName
	}
	return n.Data.FunctionName
}

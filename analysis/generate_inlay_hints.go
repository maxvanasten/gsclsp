package analysis

import (
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

func GenerateInlayHints(signatures []FunctionSignature, nodes []p.Node, tokens []l.Token) []lsp.InlayHint {
	hints := []lsp.InlayHint{}

	for _, n := range nodes {
		if n.Type == "function_call" {
			if len(n.Children) > 0 {
				// See if we can find it in signatures
				labels := []string{}
				for _, s := range signatures {
					if s.Name == n.Data.FunctionName {
						labels = append(labels, s.Arguments...)
					}
				}

				if len(labels) > 0 {
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
					if !callClosedOnLine(n, tokens) {
						paramIndex, stubCol, ok := openCallParamAnchor(n, tokens)
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
				hints = append(hints, GenerateInlayHints(signatures, n.Children, tokens)...)
			}
		}
	}

	return hints
}

func callClosedOnLine(n p.Node, tokens []l.Token) bool {
	if n.Line <= 0 || n.Data.FunctionName == "" {
		return false
	}
	for i, t := range tokens {
		if t.Type != l.SYMBOL || t.Content != n.Data.FunctionName || t.Line != n.Line {
			continue
		}
		seenOpen := false
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].Type == l.NEWLINE || tokens[j].Type == l.TERMINATOR {
				break
			}
			if tokens[j].Type == l.OPEN_PAREN {
				seenOpen = true
				continue
			}
			if tokens[j].Type == l.CLOSE_PAREN {
				return seenOpen
			}
		}
	}
	return false
}

func openCallParamAnchor(n p.Node, tokens []l.Token) (int, int, bool) {
	if n.Line <= 0 || n.Data.FunctionName == "" {
		return 0, 0, false
	}
	for i, t := range tokens {
		if t.Type != l.SYMBOL || t.Content != n.Data.FunctionName || t.Line != n.Line {
			continue
		}
		seenOpen := false
		commaCount := 0
		currentCol := 0
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].Type == l.NEWLINE || tokens[j].Type == l.TERMINATOR {
				break
			}
			if tokens[j].Type == l.OPEN_PAREN {
				seenOpen = true
				currentCol = tokens[j].EndCol + 1
				continue
			}
			if !seenOpen {
				continue
			}
			if tokens[j].Type == l.COMMA {
				commaCount++
				currentCol = tokens[j].EndCol + 1
			}
		}
		if seenOpen {
			return commaCount, currentCol - 1, true
		}
	}
	return 0, 0, false
}

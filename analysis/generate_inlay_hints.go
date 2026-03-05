package analysis

import (
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	p "github.com/maxvanasten/gscp/parser"
)

func GenerateInlayHints(signatures []FunctionSignature, nodes []p.Node) []lsp.InlayHint {
	hints := []lsp.InlayHint{}

	for _, n := range nodes {
		if n.Type == "function_call" {
			// See if we can find it in signatures
			labels := []string{}
			for _, s := range signatures {
				if s.Name == n.Data.FunctionName {
					labels = append(labels, s.Arguments...)
				}
			}

			if len(labels) > 0 {
				baseLine := n.Line
				for i, a := range n.Children {
					label := strings.Builder{}
					label.WriteString(labels[i])
					label.WriteString(": ")
					hints = append(hints, lsp.InlayHint{
						Position: lsp.Position{
							Line:      baseLine - 1,
							Character: a.Col - 1,
						},
						Label: label.String(),
					})
				}
			}
		} else {
			hints = append(hints, GenerateInlayHints(signatures, n.Children)...)
		}
	}

	return hints
}

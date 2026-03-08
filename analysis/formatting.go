package analysis

import (
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gscp/diagnostics"
	"github.com/maxvanasten/gscp/generator"
	p "github.com/maxvanasten/gscp/parser"
)

const formattingFallbackTabSize = 4

func (s *State) Formatting(id int, uri string, options lsp.FormattingOptions) lsp.DocumentFormattingResponse {
	original, ok := s.Documents[uri]
	if !ok {
		return formattingResponse(id, nil)
	}

	parseResult, err := Parse(original)
	if err != nil {
		return formattingResponse(id, nil)
	}
	if hasErrorDiagnostics(parseResult.Diagnostics) {
		return formattingResponse(id, nil)
	}

	previousIndent := generator.Indent
	generator.Indent = formattingIndent(options)
	defer func() {
		generator.Indent = previousIndent
	}()

	formattedLines := make([]string, 0, len(parseResult.Ast))
	for _, node := range parseResult.Ast {
		formattedLines = append(formattedLines, generator.Generate(node))
	}
	formatted := joinFormattedNodesWithOriginalSpacing(parseResult.Ast, formattedLines)

	if formatted == original {
		return formattingResponse(id, nil)
	}

	return formattingResponse(id, []lsp.TextEdit{{
		Range:   fullDocumentRange(original),
		NewText: formatted,
	}})
}

func formattingResponse(id int, edits []lsp.TextEdit) lsp.DocumentFormattingResponse {
	if edits == nil {
		edits = []lsp.TextEdit{}
	}
	return lsp.DocumentFormattingResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: edits,
	}
}

func hasErrorDiagnostics(items []diagnostics.Diagnostic) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Severity), "error") {
			return true
		}
	}
	return false
}

func formattingIndent(options lsp.FormattingOptions) string {
	if !options.InsertSpaces && options.TabSize > 0 {
		return "\t"
	}
	tabSize := options.TabSize
	if tabSize <= 0 {
		tabSize = formattingFallbackTabSize
	}
	return strings.Repeat(" ", tabSize)
}

func fullDocumentRange(text string) lsp.Range {
	if text == "" {
		return lsp.Range{}
	}

	lines := strings.Split(text, "\n")
	lastLine := len(lines) - 1
	lastChar := len(lines[lastLine])

	return lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: lastLine, Character: lastChar},
	}
}

func joinFormattedNodesWithOriginalSpacing(nodes []p.Node, formatted []string) string {
	if len(formatted) == 0 {
		return ""
	}
	if len(formatted) == 1 || len(nodes) != len(formatted) {
		return strings.Join(formatted, "\n")
	}

	var b strings.Builder
	for i, current := range formatted {
		if i > 0 {
			separator := "\n"
			if hasBlankLineBetweenNodes(nodes[i-1], nodes[i]) {
				separator = "\n\n"
			}
			b.WriteString(separator)
		}
		b.WriteString(current)
	}
	return b.String()
}

func hasBlankLineBetweenNodes(previous, next p.Node) bool {
	previousEnd := nodeEndLine(previous)
	nextStart := next.Line
	if previousEnd <= 0 || nextStart <= 0 {
		return false
	}
	return nextStart-previousEnd > 1
}

func nodeEndLine(node p.Node) int {
	end := node.Line
	for _, child := range node.Children {
		childEnd := nodeEndLine(child)
		if childEnd > end {
			end = childEnd
		}
	}
	return end
}

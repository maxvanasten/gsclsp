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
		formattedLines = append(formattedLines, generateNodeWithOriginalSpacing(node))
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
		currentOutput := current
		if i > 0 {
			separator := nodeSeparator(nodes[i-1], nodes[i])
			if separator == " " {
				currentOutput = strings.TrimLeft(currentOutput, " \t")
			}
			b.WriteString(separator)
		}
		b.WriteString(currentOutput)
	}
	return b.String()
}

func nodeSeparator(previous, next p.Node) string {
	if shouldInlineIfElse(previous, next) {
		return " "
	}
	if hasBlankLineBetweenNodes(previous, next) {
		return "\n\n"
	}
	return "\n"
}

func shouldInlineIfElse(previous, next p.Node) bool {
	if previous.Type == "if_statement" && (next.Type == "else_clause" || next.Type == "else_header") {
		return true
	}
	if previous.Type == "else_header" && next.Type == "if_statement" {
		return true
	}
	return false
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

func generateNodeWithOriginalSpacing(node p.Node) string {
	switch node.Type {
	case "function_declaration":
		header := strings.Builder{}
		header.WriteString(node.Data.FunctionName)
		header.WriteString("(")
		if len(node.Children) > 0 {
			header.WriteString(joinInlineChildrenWithOriginalSpacing(node.Children[0].Children, ", "))
		}
		header.WriteString(")")
		if len(node.Children) > 1 {
			return formatBlockWithOriginalSpacing(header.String(), node.Children[1])
		}
		return header.String() + "\n{\n}"
	case "scope":
		lines := make([]string, 0, len(node.Children))
		for _, child := range node.Children {
			line := generateNodeWithOriginalSpacing(child)
			if child.Type == "function_call" {
				line = ensureStatementTerminator(line)
			}
			lines = append(lines, indentMultilineForFormatting(line, generator.Indent))
		}
		return joinFormattedNodesWithOriginalSpacing(node.Children, lines)
	case "if_statement":
		condition := ""
		if len(node.Children) > 0 {
			condition = generateNodeWithOriginalSpacing(node.Children[0])
		}
		header := "if (" + condition + ")"
		if len(node.Children) > 1 {
			return formatBlockWithOriginalSpacing(header, node.Children[1])
		}
		return header + "\n{\n}"
	case "else_clause":
		if len(node.Children) > 0 {
			return formatBlockWithOriginalSpacing("else", node.Children[0])
		}
		return "else\n{\n}"
	case "else_header":
		return "else"
	case "while_loop":
		condition := ""
		if len(node.Children) > 0 {
			condition = generateNodeWithOriginalSpacing(node.Children[0])
		}
		header := "while (" + condition + ")"
		if len(node.Children) > 1 {
			return formatBlockWithOriginalSpacing(header, node.Children[1])
		}
		return header + "\n{\n}"
	case "for_loop":
		init := ""
		condition := ""
		post := ""
		if len(node.Children) > 0 {
			init = generateNodeWithOriginalSpacing(node.Children[0])
		}
		if len(node.Children) > 1 {
			condition = generateNodeWithOriginalSpacing(node.Children[1])
		}
		if len(node.Children) > 2 {
			post = generateNodeWithOriginalSpacing(node.Children[2])
		}

		header := strings.Builder{}
		if init == "" && condition == "" && post == "" {
			header.WriteString("for ( ;; )")
		} else {
			header.WriteString("for (")
			if init == "" {
				header.WriteString(" ")
			} else {
				header.WriteString(init)
			}
			header.WriteString("; ")
			if condition != "" {
				header.WriteString(condition)
			}
			header.WriteString("; ")
			if post != "" {
				header.WriteString(post)
			}
			header.WriteString(")")
		}
		if len(node.Children) > 3 {
			return formatBlockWithOriginalSpacing(header.String(), node.Children[3])
		}
		return header.String() + "\n{\n}"
	case "foreach_loop":
		vars := ""
		iter := ""
		if len(node.Children) > 0 {
			vars = generateNodeWithOriginalSpacing(node.Children[0])
		}
		if len(node.Children) > 1 {
			iter = generateNodeWithOriginalSpacing(node.Children[1])
		}
		header := "foreach (" + vars + " in " + iter + ")"
		if len(node.Children) > 2 {
			return formatBlockWithOriginalSpacing(header, node.Children[2])
		}
		return header + "\n{\n}"
	case "switch_expr":
		if len(node.Children) > 0 {
			return strings.TrimSuffix(generateNodeWithOriginalSpacing(node.Children[0]), ";")
		}
		return ""
	case "case_clause":
		label := ""
		if len(node.Children) > 0 {
			label = strings.TrimSuffix(generateNodeWithOriginalSpacing(node.Children[0]), ";")
		}
		return "case " + label + ":"
	case "default_clause":
		return "default:"
	case "switch_statement":
		switchExpr := ""
		if len(node.Children) > 0 {
			switchExpr = strings.TrimSuffix(generateNodeWithOriginalSpacing(node.Children[0]), ";")
		}

		var b strings.Builder
		b.WriteString("switch(")
		b.WriteString(switchExpr)
		b.WriteString(") {")

		if len(node.Children) > 1 {
			if scopeBody := formatSwitchScopeWithOriginalSpacing(node.Children[1]); scopeBody != "" {
				b.WriteString("\n")
				b.WriteString(scopeBody)
			}
		}

		b.WriteString("\n}")
		return b.String()
	default:
		return generator.Generate(node)
	}
}

func formatSwitchScopeWithOriginalSpacing(scope p.Node) string {
	lines := make([]string, 0, len(scope.Children))
	inCase := false

	for _, child := range scope.Children {
		line := generateNodeWithOriginalSpacing(child)
		if child.Type == "function_call" {
			line = ensureStatementTerminator(line)
		}

		if child.Type == "case_clause" || child.Type == "default_clause" {
			inCase = true
			lines = append(lines, indentMultilineForFormatting(line, generator.Indent))
			continue
		}

		if inCase {
			lines = append(lines, indentMultilineForFormatting(line, generator.Indent+generator.Indent))
			continue
		}

		lines = append(lines, indentMultilineForFormatting(line, generator.Indent))
	}

	return joinFormattedNodesWithOriginalSpacing(scope.Children, lines)
}

func formatBlockWithOriginalSpacing(header string, scope p.Node) string {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n{")
	if scopeBody := generateNodeWithOriginalSpacing(scope); scopeBody != "" {
		b.WriteString("\n")
		b.WriteString(scopeBody)
	}
	b.WriteString("\n}")
	return b.String()
}

func joinInlineChildrenWithOriginalSpacing(children []p.Node, separator string) string {
	parts := make([]string, 0, len(children))
	for i := 0; i < len(children); i++ {
		child := children[i]
		if child.Type == "variable_reference" && child.Data.VarName == "#" && i+1 < len(children) {
			next := children[i+1]
			if next.Type == "string" {
				parts = append(parts, "#\""+next.Data.Content+"\"")
				i++
				continue
			}
		}
		parts = append(parts, strings.TrimSuffix(generateNodeWithOriginalSpacing(child), ";"))
	}
	return strings.Join(parts, separator)
}

func indentMultilineForFormatting(value, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = prefix
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func ensureStatementTerminator(line string) string {
	if strings.HasSuffix(strings.TrimRight(line, " \t\r\n"), ";") {
		return line
	}
	return line + ";"
}

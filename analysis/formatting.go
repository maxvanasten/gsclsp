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
	s.mu.RLock()
	original, ok := s.Documents[uri]
	s.mu.RUnlock()
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

	originalLines := strings.Split(original, "\n")
	formattedLines := make([]string, 0, len(parseResult.Ast))
	for _, node := range parseResult.Ast {
		formattedLines = append(formattedLines, generateNodeWithOriginalSpacingAndLines(node, originalLines))
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
			line = collapseDuplicateStatementTerminators(line)
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
			return formatBlockWithInlineOpeningBrace(header, node.Children[1])
		}
		return header + " {\n}"
	case "else_clause":
		if len(node.Children) > 0 {
			return formatBlockWithInlineOpeningBrace("else", node.Children[0])
		}
		return "else {\n}"
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
	case "return_statement":
		if len(node.Children) == 0 {
			return "return;"
		}
		value := strings.TrimSuffix(generateNodeWithOriginalSpacing(node.Children[0]), ";")
		if value == "" {
			return "return;"
		}
		return "return " + value + ";"
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
		line = collapseDuplicateStatementTerminators(line)

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

func formatBlockWithInlineOpeningBrace(header string, scope p.Node) string {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString(" {")
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

func collapseDuplicateStatementTerminators(line string) string {
	trimmed := strings.TrimRight(line, " \t\r\n")
	if !strings.HasSuffix(trimmed, ";;") {
		return line
	}
	for strings.HasSuffix(trimmed, ";;") {
		trimmed = strings.TrimSuffix(trimmed, ";")
	}
	return trimmed + ";"
}

func generateNodeWithOriginalSpacingAndLines(node p.Node, originalLines []string) string {
	// Handle scope specially to detect function pointer patterns
	if node.Type == "scope" {
		return formatScopeWithLines(node, originalLines)
	}

	// For function declarations with a scope body, process the body with lines
	if node.Type == "function_declaration" && len(node.Children) > 1 {
		header := strings.Builder{}
		header.WriteString(node.Data.FunctionName)
		header.WriteString("(")
		if len(node.Children) > 0 && len(node.Children[0].Children) > 0 {
			header.WriteString(joinInlineChildrenWithOriginalSpacing(node.Children[0].Children, ", "))
		}
		header.WriteString(")")

		bodyScope := node.Children[1]
		bodyContent := formatScopeWithLines(bodyScope, originalLines)

		// Reconstruct: header + newline + { + body + newline + }
		var result strings.Builder
		result.WriteString(header.String())
		result.WriteString("\n{")
		if bodyContent != "" {
			result.WriteString("\n")
			result.WriteString(bodyContent)
		}
		result.WriteString("\n}")
		return result.String()
	}

	return generateNodeWithOriginalSpacing(node)
}

func formatScopeWithLines(node p.Node, originalLines []string) string {
	lines := make([]string, 0, len(node.Children))
	for i := 0; i < len(node.Children); i++ {
		child := node.Children[i]

		// Check if this and subsequent nodes form a function pointer pattern (thread case)
		if formatted, skipCount, ok := tryFormatFunctionPointerPattern(node.Children, i, originalLines); ok {
			lines = append(lines, indentMultilineForFormatting(formatted, generator.Indent))
			i += skipCount
			continue
		}

		// Check if a function_call on this line has function pointer syntax (non-thread case)
		if child.Type == "function_call" {
			if child.Line > 0 && child.Line <= len(originalLines) {
				originalLine := originalLines[child.Line-1]
				if containsFunctionPointerSyntax(originalLine) {
					statement := extractStatementFromLineWithCol(originalLine, child.Col, child)
					if statement != "" {
						lines = append(lines, indentMultilineForFormatting(statement, generator.Indent))
						continue
					}
				}
			}
		}

		line := generateNodeWithOriginalSpacing(child)
		if child.Type == "function_call" {
			line = ensureStatementTerminator(line)
		}
		line = collapseDuplicateStatementTerminators(line)
		lines = append(lines, indentMultilineForFormatting(line, generator.Indent))
	}
	return joinFormattedNodesWithOriginalSpacing(node.Children, lines)
}

func tryFormatFunctionPointerPattern(children []p.Node, startIdx int, originalLines []string) (string, int, bool) {
	if startIdx >= len(children) {
		return "", 0, false
	}

	first := children[startIdx]
	firstLine := first.Line
	if firstLine <= 0 || firstLine > len(originalLines) {
		return "", 0, false
	}

	originalLine := originalLines[firstLine-1]

	// Check for function pointer syntax: method thread [[ expr ]]() or method [[ expr ]]()
	// Pattern: variable_reference/thread_keyword/array_literal sequence on same line
	if first.Type != "variable_reference" {
		return "", 0, false
	}

	// Check if this line contains function pointer pattern in original text
	if !containsFunctionPointerSyntax(originalLine) {
		return "", 0, false
	}

	// Count how many nodes are part of this pattern (all on the same line)
	endIdx := startIdx
	for i := startIdx + 1; i < len(children); i++ {
		if children[i].Line == firstLine {
			endIdx = i
		} else {
			break
		}
	}

	// Extract the statement using column information from nodes
	statement := extractStatementFromLineWithCol(originalLine, first.Col, children[endIdx])
	if statement == "" {
		return "", 0, false
	}

	return statement, endIdx - startIdx, true
}

func extractStatementFromLineWithCol(line string, startCol int, lastNode p.Node) string {
	// Use 0-indexed column (AST uses 1-indexed)
	start := startCol - 1
	if start < 0 {
		start = 0
	}
	if start >= len(line) {
		return ""
	}

	// Find the end position - need to include trailing (); if present
	// The AST nodes may not include the () and ;, so look for them in the line
	nodeEnd := lastNode.Col - 1 + lastNode.Length
	if nodeEnd <= start {
		nodeEnd = start
	}

	// Look for the statement terminator after the node end
	searchStart := nodeEnd
	if searchStart > len(line) {
		searchStart = len(line)
	}

	// Find the semicolon to get the full statement including ();
	semicolonIdx := strings.Index(line[searchStart:], ";")
	if semicolonIdx >= 0 {
		end := searchStart + semicolonIdx + 1 // Include the semicolon
		if end <= len(line) {
			return line[start:end]
		}
	}

	// If no semicolon found, use the node end position
	if nodeEnd > len(line) {
		nodeEnd = len(line)
	}

	return line[start:nodeEnd]
}

func containsFunctionPointerSyntax(line string) bool {
	// Check for [[...]] pattern followed by ()
	doubleBracketStart := strings.Index(line, "[[")
	if doubleBracketStart < 0 {
		return false
	}
	doubleBracketEnd := strings.Index(line[doubleBracketStart:], "]]")
	if doubleBracketEnd < 0 {
		return false
	}
	doubleBracketEnd += doubleBracketStart

	// Check for () after ]]
	afterBrackets := line[doubleBracketEnd+2:]
	parenStart := strings.Index(afterBrackets, "(")
	if parenStart < 0 {
		return false
	}

	// Check that there's a closing paren
	afterOpenParen := afterBrackets[parenStart+1:]
	if !strings.Contains(afterOpenParen, ")") {
		return false
	}

	return true
}

func extractStatementFromLine(line string) string {
	// Trim leading whitespace but preserve the statement
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return ""
	}

	// Find the statement terminator
	semicolonIdx := strings.Index(trimmed, ";")
	if semicolonIdx >= 0 {
		return trimmed[:semicolonIdx+1]
	}

	// If no semicolon, return the whole trimmed line
	return trimmed
}

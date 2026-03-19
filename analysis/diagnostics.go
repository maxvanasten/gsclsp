package analysis

import (
	"fmt"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gscp/diagnostics"
	lex "github.com/maxvanasten/gscp/lexer"
)

const diagnosticsSource = "gscp"

func toLspDiagnostics(items []diagnostics.Diagnostic, tokens []lex.Token) []lsp.Diagnostic {
	result := make([]lsp.Diagnostic, 0, len(items))
	for _, d := range items {
		// Filter out false positive "unclosed parenthesis" errors in comments
		if isFalsePositiveUnclosedParenDiagnostic(d, tokens) {
			continue
		}

		startLine, startCol := toZeroBasedPosition(d.Line, d.Col)
		endLine, endCol := toZeroBasedPosition(d.EndLine, d.EndCol)
		if endLine < startLine || (endLine == startLine && endCol < startCol) {
			endLine = startLine
			endCol = startCol
		}
		result = append(result, lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{Line: startLine, Character: startCol},
				End:   lsp.Position{Line: endLine, Character: endCol},
			},
			Severity: mapSeverity(d.Severity),
			Source:   diagnosticsSource,
			Message:  d.Message,
		})
	}
	return result
}

// isFalsePositiveUnclosedParenDiagnostic checks if a diagnostic about unclosed parens
// is a false positive - i.e., the parens are actually inside a comment
func isFalsePositiveUnclosedParenDiagnostic(d diagnostics.Diagnostic, tokens []lex.Token) bool {
	msg := strings.ToLower(d.Message)
	// Check if this is a diagnostic about unclosed parentheses
	if !strings.Contains(msg, "unclosed") || (!strings.Contains(msg, "(") && !strings.Contains(msg, ")")) {
		return false
	}

	// If no tokens available, we can't determine context - don't filter
	if len(tokens) == 0 {
		return false
	}

	// Check if there's a comment token at the diagnostic position
	for _, tok := range tokens {
		// Check if token is on the same line as the diagnostic
		if tok.Line == d.Line {
			// Check if this is a comment token and the diagnostic column falls within it
			if tok.Type == lex.LINE_COMMENT || tok.Type == lex.BLOCK_COMMENT {
				startCol := tok.Col
				endCol := tok.EndCol
				if endCol < startCol {
					endCol = startCol + len(tok.Content)
				}
				// If diagnostic is within or near this comment, consider it a false positive
				if d.Col >= startCol && d.Col <= endCol {
					return true
				}
			}
		}
	}

	return false
}

func toZeroBasedPosition(line, col int) (int, int) {
	if line <= 0 {
		line = 1
	}
	if col <= 0 {
		col = 1
	}
	return line - 1, col - 1
}

func mapSeverity(severity string) lsp.DiagnosticSeverity {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error":
		return lsp.DiagnosticSeverityError
	case "warning", "warn":
		return lsp.DiagnosticSeverityWarning
	case "information", "info":
		return lsp.DiagnosticSeverityInformation
	case "hint":
		return lsp.DiagnosticSeverityHint
	default:
		return 0
	}
}

func parseFailureDiagnostic(err error) lsp.Diagnostic {
	msg := fmt.Sprintf("parser error: %v", err)
	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}
	return lsp.Diagnostic{
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 0},
		},
		Severity: lsp.DiagnosticSeverityError,
		Source:   "gsclsp",
		Message:  msg,
	}
}

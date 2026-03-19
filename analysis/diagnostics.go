package analysis

import (
	"fmt"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gscp/diagnostics"
)

const diagnosticsSource = "gscp"

func toLspDiagnostics(items []diagnostics.Diagnostic) []lsp.Diagnostic {
	result := make([]lsp.Diagnostic, 0, len(items))
	for _, d := range items {
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

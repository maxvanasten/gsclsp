package analysis

import (
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gscp/diagnostics"
	lex "github.com/maxvanasten/gscp/lexer"
)

func TestIsFalsePositiveUnclosedParenDiagnostic(t *testing.T) {
	tests := []struct {
		name     string
		diag     diagnostics.Diagnostic
		tokens   []lex.Token
		expected bool
	}{
		{
			name: "not unclosed paren message",
			diag: diagnostics.Diagnostic{
				Message: "undefined variable",
				Line:    1,
				Col:     10,
			},
			tokens:   []lex.Token{},
			expected: false,
		},
		{
			name: "unclosed paren in code (not comment)",
			diag: diagnostics.Diagnostic{
				Message: "unclosed (",
				Line:    1,
				Col:     10,
			},
			tokens: []lex.Token{
				{Type: lex.SYMBOL, Line: 1, Col: 1, Content: "foo"},
			},
			expected: false,
		},
		{
			name: "unclosed paren in comment - should be filtered",
			diag: diagnostics.Diagnostic{
				Message: "unclosed )",
				Line:    1,
				Col:     35,
			},
			tokens: []lex.Token{
				{Type: lex.SYMBOL, Line: 1, Col: 1, Content: "test"},
				{Type: lex.LINE_COMMENT, Line: 1, Col: 30, EndCol: 50, Content: "// comment ("},
			},
			expected: true,
		},
		{
			name: "unclosed paren in block comment - should be filtered",
			diag: diagnostics.Diagnostic{
				Message: "unclosed (",
				Line:    2,
				Col:     15,
			},
			tokens: []lex.Token{
				{Type: lex.BLOCK_COMMENT, Line: 2, Col: 10, EndCol: 30, Content: "/* comment () */"},
			},
			expected: true,
		},
		{
			name: "no tokens available - don't filter",
			diag: diagnostics.Diagnostic{
				Message: "unclosed )",
				Line:    1,
				Col:     10,
			},
			tokens:   []lex.Token{},
			expected: false,
		},
		{
			name: "diagnostic outside comment range",
			diag: diagnostics.Diagnostic{
				Message: "unclosed )",
				Line:    1,
				Col:     5,
			},
			tokens: []lex.Token{
				{Type: lex.LINE_COMMENT, Line: 1, Col: 30, EndCol: 50, Content: "// comment ("},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFalsePositiveUnclosedParenDiagnostic(tt.diag, tt.tokens)
			if result != tt.expected {
				t.Errorf("isFalsePositiveUnclosedParenDiagnostic() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToLspDiagnostics_FiltersFalsePositives(t *testing.T) {
	items := []diagnostics.Diagnostic{
		{
			Message:  "real error",
			Line:     1,
			Col:      1,
			EndLine:  1,
			EndCol:   5,
			Severity: "error",
		},
		{
			Message:  "unclosed ) in comment",
			Line:     2,
			Col:      35,
			EndLine:  2,
			EndCol:   36,
			Severity: "error",
		},
	}

	tokens := []lex.Token{
		{Type: lex.LINE_COMMENT, Line: 2, Col: 30, EndCol: 50, Content: "// comment )"},
	}

	result := toLspDiagnostics(items, tokens)

	if len(result) != 1 {
		t.Errorf("expected 1 diagnostic after filtering, got %d", len(result))
	}

	if result[0].Message != "real error" {
		t.Errorf("expected 'real error', got '%s'", result[0].Message)
	}
}

func TestToLspDiagnostics_PreservesAllDiagnosticsWithoutTokens(t *testing.T) {
	items := []diagnostics.Diagnostic{
		{
			Message:  "error 1",
			Line:     1,
			Col:      1,
			EndLine:  1,
			EndCol:   5,
			Severity: "error",
		},
		{
			Message:  "error 2",
			Line:     2,
			Col:      10,
			EndLine:  2,
			EndCol:   15,
			Severity: "warning",
		},
	}

	result := toLspDiagnostics(items, nil)

	if len(result) != 2 {
		t.Errorf("expected 2 diagnostics, got %d", len(result))
	}
}

func TestToLspDiagnostics_MapsSeverityCorrectly(t *testing.T) {
	tests := []struct {
		input    string
		expected lsp.DiagnosticSeverity
	}{
		{"error", lsp.DiagnosticSeverityError},
		{"ERROR", lsp.DiagnosticSeverityError},
		{"warning", lsp.DiagnosticSeverityWarning},
		{"warn", lsp.DiagnosticSeverityWarning},
		{"information", lsp.DiagnosticSeverityInformation},
		{"info", lsp.DiagnosticSeverityInformation},
		{"hint", lsp.DiagnosticSeverityHint},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			items := []diagnostics.Diagnostic{
				{
					Message:  "test",
					Line:     1,
					Col:      1,
					Severity: tt.input,
				},
			}

			result := toLspDiagnostics(items, nil)

			if len(result) != 1 {
				t.Fatalf("expected 1 result, got %d", len(result))
			}

			if result[0].Severity != tt.expected {
				t.Errorf("severity = %v, want %v", result[0].Severity, tt.expected)
			}
		})
	}
}

func TestToZeroBasedPosition(t *testing.T) {
	tests := []struct {
		line, col       int
		expLine, expCol int
	}{
		{1, 1, 0, 0},
		{5, 10, 4, 9},
		{0, 0, 0, 0},
		{-1, -1, 0, 0},
		{10, 0, 9, 0},
	}

	for _, tt := range tests {
		l, c := toZeroBasedPosition(tt.line, tt.col)
		if l != tt.expLine || c != tt.expCol {
			t.Errorf("toZeroBasedPosition(%d, %d) = (%d, %d), want (%d, %d)",
				tt.line, tt.col, l, c, tt.expLine, tt.expCol)
		}
	}
}

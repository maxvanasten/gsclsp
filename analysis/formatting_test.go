package analysis

import (
	"strings"
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
)

func TestFormattingReturnsEditsForUnformattedDocument(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){wait 0.05;}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(1, uri, lsp.FormattingOptions{TabSize: 2, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if response.Result[0].NewText == state.Documents[uri] {
		t.Fatal("expected formatting output to differ from original")
	}
}

func TestFormattingUsesRequestedSpaceIndentWidth(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(1){wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(4, uri, lsp.FormattingOptions{TabSize: 8, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if !strings.Contains(response.Result[0].NewText, "\n        if (1)\n") {
		t.Fatalf("expected 8-space indentation, got: %q", response.Result[0].NewText)
	}
}

func TestFormattingUsesFallbackSpaceIndentWidth(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(1){wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(5, uri, lsp.FormattingOptions{})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if !strings.Contains(response.Result[0].NewText, "\n    if (1)\n") {
		t.Fatalf("expected fallback 4-space indentation, got: %q", response.Result[0].NewText)
	}
}

func TestFormattingUsesTabsWhenInsertSpacesDisabled(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(1){wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(6, uri, lsp.FormattingOptions{InsertSpaces: false, TabSize: 8})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if !strings.Contains(response.Result[0].NewText, "\n\tif (1)\n") {
		t.Fatalf("expected tab indentation, got: %q", response.Result[0].NewText)
	}
}

func TestFormattingPreservesComments(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\n// keep line\nwait 0.05;\n/# keep block #/\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(7, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "// keep line") {
		t.Fatalf("expected line comment to be preserved, got: %q", formatted)
	}
	if !strings.Contains(formatted, "/# keep block #/") {
		t.Fatalf("expected block comment to be preserved, got: %q", formatted)
	}
}

func TestFormattingReturnsNoEditsOnParseFailure(t *testing.T) {
	t.Setenv("PATH", "")
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){wait 0.05;}"

	response := state.Formatting(2, uri, lsp.FormattingOptions{})
	if len(response.Result) != 0 {
		t.Fatalf("expected no edits on parse failure, got %d", len(response.Result))
	}
}

func TestFormattingReturnsNoEditsForMissingDocument(t *testing.T) {
	state := NewState()
	response := state.Formatting(3, "file:///tmp/missing.gsc", lsp.FormattingOptions{})
	if len(response.Result) != 0 {
		t.Fatalf("expected no edits for missing document, got %d", len(response.Result))
	}
}

func ensureParserAvailable(t *testing.T, input string) {
	t.Helper()
	if _, err := Parse(input); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			t.Skip("gscp not available on PATH")
		}
		t.Fatalf("parse precheck failed: %v", err)
	}
}

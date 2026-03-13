package analysis

import (
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
)

func TestApplyIncrementalChange_InsertText(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() {}"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "test",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 6},
			End:   lsp.Position{Line: 0, Character: 6},
		},
	})

	expected := "main()test {}"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_DeleteText(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() test {}"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 7},
			End:   lsp.Position{Line: 0, Character: 11},
		},
	})

	expected := "main()  {}"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_ReplaceText(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() old {}"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "new",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 7},
			End:   lsp.Position{Line: 0, Character: 10},
		},
	})

	expected := "main() new {}"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_MultilineInsert(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "line1\nline2\nline3"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "inserted\nbetween",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 1, Character: 0},
			End:   lsp.Position{Line: 1, Character: 0},
		},
	})

	expected := "line1\ninserted\nbetweenline2\nline3"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_MultilineDelete(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "line1\nline2\nline3"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 5},
			End:   lsp.Position{Line: 2, Character: 0},
		},
	})

	expected := "line1line3"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_FullReplacementFallback(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "original"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "full replacement",
	})

	expected := "full replacement"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_FullReplacementFallbackNilRange(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "original"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text:  "full replacement",
		Range: nil,
	})

	expected := "full replacement"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_OutOfBoundsFallback(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() {}"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "fallback text",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 100, Character: 0},
			End:   lsp.Position{Line: 100, Character: 5},
		},
	})

	expected := "fallback text"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

func TestApplyIncrementalChange_MultipleChangesInSequence(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() {}"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "test",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 6},
			End:   lsp.Position{Line: 0, Character: 6},
		},
	})

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: "other",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 10},
			End:   lsp.Position{Line: 0, Character: 10},
		},
	})

	expected := "main()testother {}"
	if state.Documents[uri] != expected {
		t.Errorf("expected %q, got %q", expected, state.Documents[uri])
	}
}

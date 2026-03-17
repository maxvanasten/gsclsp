package analysis

import (
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
)

func TestEnsureParsed_DoesNotParseWhenClean(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = false

	state.EnsureParsed(uri)

	if len(state.Ast) != 0 {
		t.Error("EnsureParsed should not parse when not dirty")
	}
}

func TestEnsureParsed_ParsesWhenDirty(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = true

	state.EnsureParsed(uri)

	if len(state.Ast[uri]) == 0 {
		t.Error("EnsureParsed should parse when dirty")
	}
	if state.AstDirty[uri] {
		t.Error("AstDirty should be false after EnsureParsed")
	}
}

func TestApplyIncrementalChange_MarksDirty(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() {}"

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: " test",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 6},
			End:   lsp.Position{Line: 0, Character: 6},
		},
	})

	if !state.AstDirty[uri] {
		t.Error("ApplyIncrementalChange should mark document as dirty")
	}
}

func TestApplyIncrementalChange_ClearsDiagnostics(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() {}"
	state.Diagnostics[uri] = []lsp.Diagnostic{{Message: "error"}}

	state.ApplyIncrementalChange(uri, lsp.TextDocumentContentChangeEvent{
		Text: " test",
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 6},
			End:   lsp.Position{Line: 0, Character: 6},
		},
	})

	if state.Diagnostics[uri] != nil {
		t.Error("Diagnostics should be cleared after incremental change")
	}
}

func TestClearCaches_ClearsResolvedAndIncludeOrigins(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Resolved[uri] = resolvedCacheEntry{
		Signatures:      []FunctionSignature{{Name: "test"}},
		IncludeModTimes: map[string]int64{},
	}
	state.IncludeOrigins[uri] = includeOriginsCacheEntry{
		Origins:         map[string]string{"foo": "bar"},
		IncludeModTimes: map[string]int64{},
	}

	state.ClearCaches(uri)

	if _, ok := state.Resolved[uri]; ok {
		t.Error("Resolved should be cleared")
	}
	if _, ok := state.IncludeOrigins[uri]; ok {
		t.Error("IncludeOrigins should be cleared")
	}
}

func TestLazyParsing_HoverTriggersParse(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = true

	_ = state.Hover(1, uri, lsp.Position{Line: 0, Character: 0})

	if state.AstDirty[uri] {
		t.Error("Hover should trigger parse and clear dirty flag")
	}
	if len(state.Ast[uri]) == 0 {
		t.Error("Hover should populate AST")
	}
}

func TestLazyParsing_InlayHintsTriggersParse(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = true

	_ = state.InlayHints(1, uri)

	if state.AstDirty[uri] {
		t.Error("InlayHints should trigger parse and clear dirty flag")
	}
	if len(state.Ast[uri]) == 0 {
		t.Error("InlayHints should populate AST")
	}
}

func TestLazyParsing_DefinitionTriggersParse(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = true

	_ = state.Definition(1, uri, lsp.Position{Line: 0, Character: 0})

	if state.AstDirty[uri] {
		t.Error("Definition should trigger parse and clear dirty flag")
	}
	if len(state.Ast[uri]) == 0 {
		t.Error("Definition should populate AST")
	}
}

func TestLazyParsing_CompletionTriggersParse(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = true

	_ = state.Completion(1, uri, lsp.Position{Line: 0, Character: 0})

	if state.AstDirty[uri] {
		t.Error("Completion should trigger parse and clear dirty flag")
	}
	if len(state.Ast[uri]) == 0 {
		t.Error("Completion should populate AST")
	}
}

func TestLazyParsing_SemanticTokensTriggersParse(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/test.gsc"

	state.Documents[uri] = "main() {}"
	state.AstDirty[uri] = true

	_ = state.SemanticTokens(1, uri)

	if state.AstDirty[uri] {
		t.Error("SemanticTokens should trigger parse and clear dirty flag")
	}
	if len(state.Tokens[uri]) == 0 {
		t.Error("SemanticTokens should populate Tokens")
	}
}

package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
)

func TestCompletionFunctionsAndKeywords(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() { wa }"
	state.Signatures[uri] = []FunctionSignature{
		{Name: "wait", Arguments: []string{"duration"}},
		{Name: "wait", Arguments: []string{"duration"}},
		{Name: "waittill", Arguments: []string{"event", "entity", "data"}},
	}

	resp := state.Completion(1, uri, lsp.Position{Line: 0, Character: 11})
	wait := completionItemByLabel(resp.Result.Items, "wait")
	if wait == nil {
		t.Fatal("expected wait completion item")
	}
	if wait.Kind != lsp.CompletionItemKindFunction {
		t.Fatalf("expected wait to be function kind, got %d", wait.Kind)
	}
	if wait.InsertText != "wait(${1:duration})" {
		t.Fatalf("unexpected wait insert text: %q", wait.InsertText)
	}
	if wait.InsertTextFormat != lsp.InsertTextFormatSnippet {
		t.Fatalf("expected snippet insertTextFormat, got %d", wait.InsertTextFormat)
	}

	waittill := completionItemByLabel(resp.Result.Items, "waittill")
	if waittill == nil {
		t.Fatal("expected waittill completion item")
	}
	if waittill.InsertText != "waittill(${1:event}, ${2:entity}, ${3:data})" {
		t.Fatalf("unexpected waittill insert text: %q", waittill.InsertText)
	}
}

func TestCompletionKeywordsOnly(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main() { fo }"

	resp := state.Completion(2, uri, lsp.Position{Line: 0, Character: 11})
	item := completionItemByLabel(resp.Result.Items, "for")
	if item == nil {
		t.Fatal("expected keyword completion for for")
	}
	if item.Kind != lsp.CompletionItemKindKeyword {
		t.Fatalf("expected keyword kind, got %d", item.Kind)
	}
}

func TestCompletionIncludePathFromStdlib(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	state.Documents[uri] = "#include common_scripts\\u"

	resp := state.Completion(3, uri, lsp.Position{Line: 0, Character: len("#include common_scripts\\u")})
	item := completionItemByLabel(resp.Result.Items, "common_scripts\\utility")
	if item == nil {
		t.Fatal("expected stdlib include path completion")
	}
	if item.Kind != lsp.CompletionItemKindModule {
		t.Fatalf("expected module kind, got %d", item.Kind)
	}
}

func TestCompletionIncludePathFromLocalFiles(t *testing.T) {
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.gsc")
	helperPath := filepath.Join(dir, "helpers", "util.gsc")

	if err := os.MkdirAll(filepath.Dir(helperPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("main() { }\n"), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := os.WriteFile(helperPath, []byte("util() { }\n"), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	uri := uriForPath(mainPath)
	state.Documents[uri] = "#include hel"

	resp := state.Completion(4, uri, lsp.Position{Line: 0, Character: len("#include hel")})
	item := completionItemByLabel(resp.Result.Items, "helpers\\util")
	if item == nil {
		t.Fatal("expected local include path completion")
	}
}

func TestCompletionQualifiedFunctionFromStdlib(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	state.Documents[uri] = "main() { common_scripts\\utility::arr }"

	resp := state.Completion(5, uri, lsp.Position{Line: 0, Character: len("main() { common_scripts\\utility::arr")})
	item := completionItemByLabel(resp.Result.Items, "array_copy")
	if item == nil {
		t.Fatal("expected qualified stdlib function completion")
	}
	if item.Kind != lsp.CompletionItemKindFunction {
		t.Fatalf("expected function kind, got %d", item.Kind)
	}
}

func TestCompletionQualifiedPath(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	state.Documents[uri] = "main() { common_scripts\\ut }"

	resp := state.Completion(6, uri, lsp.Position{Line: 0, Character: len("main() { common_scripts\\ut")})
	item := completionItemByLabel(resp.Result.Items, "common_scripts\\utility")
	if item == nil {
		t.Fatal("expected qualified path completion")
	}
	if item.Kind != lsp.CompletionItemKindModule {
		t.Fatalf("expected module kind, got %d", item.Kind)
	}
}

func TestCompletionPrefixAtPosition(t *testing.T) {
	doc := "main() { array_c }"
	prefix := completionPrefixAtPosition(doc, lsp.Position{Line: 0, Character: 16})
	if prefix != "array_c" {
		t.Fatalf("expected prefix array_c, got %q", prefix)
	}
}

func TestFunctionCompletionSnippetSanitizesPlaceholder(t *testing.T) {
	snippet := functionCompletionSnippet(FunctionSignature{Name: "doit", Arguments: []string{"$name}", "  "}})
	if snippet != "doit(${1:name}, ${2:arg})" {
		t.Fatalf("unexpected snippet: %q", snippet)
	}
}

func completionItemByLabel(items []lsp.CompletionItem, label string) *lsp.CompletionItem {
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	return nil
}

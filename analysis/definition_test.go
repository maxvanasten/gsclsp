package analysis

import (
	"path/filepath"
	"testing"
)

func TestDefinitionFindsLocalDeclaration(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "foo() { }\n" +
		"main() { foo(); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 1, "foo")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected definition location")
	}
	if response.Result.URI != uri {
		t.Fatalf("unexpected uri: %s", response.Result.URI)
	}
	if response.Result.Range.Start.Line != 0 {
		t.Fatalf("unexpected start line: %d", response.Result.Range.Start.Line)
	}
}

func TestDefinitionFindsIncludedLocalDeclaration(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "test.gsc")
	helperPath := filepath.Join(dir, "helpers.gsc")

	writeFile(t, helperPath, "helpers( foo ) { }\n")
	text := "#include helpers;\n" +
		"main() { helpers(bar); }\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, text)
	position := positionForLine(text, 1, "helpers")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected definition location")
	}
	if response.Result.URI != uriForPath(helperPath) {
		t.Fatalf("unexpected uri: %s", response.Result.URI)
	}
}

func TestDefinitionReturnsNilWhenUnknown(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { missing_fn(); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 0, "missing_fn")
	response := state.Definition(1, uri, position)
	if response.Result != nil {
		t.Fatalf("expected nil result, got %+v", *response.Result)
	}
}

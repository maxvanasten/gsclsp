package analysis

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDefinitionNestedIncludeResolves(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.gsc")
	aPath := filepath.Join(dir, "a.gsc")
	bPath := filepath.Join(dir, "b.gsc")

	writeFile(t, bPath, "deep_helper( value ) { }\n")
	writeFile(t, aPath, "#include b;\n")
	mainText := "#include a;\n" +
		"main() { deep_helper(1); }\n"
	writeFile(t, mainPath, mainText)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, mainText)
	position := positionForLine(mainText, 1, "deep_helper")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected nested include definition location")
	}
	if response.Result.URI != uriForPath(bPath) {
		t.Fatalf("unexpected uri: %s", response.Result.URI)
	}
}

func TestDefinitionPrefersLocalOverIncluded(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.gsc")
	helperPath := filepath.Join(dir, "helpers.gsc")

	writeFile(t, helperPath, "dup_fn( from_include ) { }\n")
	mainText := "#include helpers;\n" +
		"dup_fn( from_local ) { }\n" +
		"main() { dup_fn(1); }\n"
	writeFile(t, mainPath, mainText)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, mainText)
	position := positionForLine(mainText, 2, "dup_fn")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected local definition location")
	}
	if response.Result.URI != uri {
		t.Fatalf("expected local uri %s, got %s", uri, response.Result.URI)
	}
	if response.Result.Range.Start.Line != 1 {
		t.Fatalf("expected local declaration line 1, got %d", response.Result.Range.Start.Line)
	}
}

func TestDefinitionQualifiedMissingIncludeReturnsNil(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { does_not_exist\\helpers::target_fn(); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 0, "target_fn")
	response := state.Definition(1, uri, position)
	if response.Result != nil {
		t.Fatalf("expected nil result, got %+v", *response.Result)
	}
}

func TestDefinitionAmbiguousIncludedOrderIsDeterministic(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.gsc")
	firstPath := filepath.Join(dir, "first.gsc")
	secondPath := filepath.Join(dir, "second.gsc")

	writeFile(t, firstPath, "dup_fn( first_arg ) { }\n")
	writeFile(t, secondPath, "dup_fn( second_arg ) { }\n")
	mainText := "#include first;\n" +
		"#include second;\n" +
		"main() { dup_fn(1); }\n"
	writeFile(t, mainPath, mainText)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, mainText)
	position := positionForLine(mainText, 2, "dup_fn")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected deterministic definition location")
	}
	if response.Result.URI != uriForPath(firstPath) {
		t.Fatalf("expected first include to win, got %s", response.Result.URI)
	}
}

func TestDefinitionFindsIncludedStdlibDeclaration(t *testing.T) {
	requireGscp(t)
	state := NewState()
	state.stdlib = map[string]map[string][]FunctionSignature{
		"mp": {
			"common_scripts/utility": {{Name: "array_copy", Arguments: []string{"array"}}},
		},
	}
	state.stdlibLoaded = true
	state.stdlibDecls = map[string]map[string][]StdlibDeclaration{
		"mp": {
			"common_scripts/utility": {{
				Name:        "array_copy",
				Arguments:   []string{"array"},
				Declaration: "array_copy( array ) {\n\treturn array;\n}",
			}},
		},
	}
	state.stdlibDeclsOk = true

	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "#include common_scripts\\utility;\n" +
		"main() { array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 1, "array_copy")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected stdlib definition location")
	}
	if !strings.Contains(response.Result.URI, "gsclsp-stdlib-defs-") {
		t.Fatalf("expected temp stdlib definition uri, got %s", response.Result.URI)
	}
	if response.Result.Range.Start.Line != 0 {
		t.Fatalf("expected declaration on first line, got %d", response.Result.Range.Start.Line)
	}
	if response.Result.Range.Start.Character != 0 {
		t.Fatalf("expected declaration at first character, got %d", response.Result.Range.Start.Character)
	}
	definitionPath := uriToPath(response.Result.URI)
	contents, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatalf("expected generated stdlib definition file: %v", err)
	}
	if !strings.Contains(string(contents), "return array;") {
		t.Fatalf("expected declaration body to be present, got %q", string(contents))
	}
}

func TestDefinitionFindsQualifiedStdlibDeclaration(t *testing.T) {
	requireGscp(t)
	state := NewState()
	state.stdlib = map[string]map[string][]FunctionSignature{
		"mp": {
			"common_scripts/utility": {{Name: "array_copy", Arguments: []string{"array"}}},
		},
	}
	state.stdlibLoaded = true
	state.stdlibDecls = map[string]map[string][]StdlibDeclaration{
		"mp": {
			"common_scripts/utility": {{
				Name:        "array_copy",
				Arguments:   []string{"array"},
				Declaration: "array_copy( array ) {\n\treturn array;\n}",
			}},
		},
	}
	state.stdlibDeclsOk = true

	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { common_scripts\\utility::array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 0, "array_copy")
	response := state.Definition(1, uri, position)
	if response.Result == nil {
		t.Fatal("expected qualified stdlib definition location")
	}
	if !strings.Contains(response.Result.URI, "gsclsp-stdlib-defs-") {
		t.Fatalf("expected temp stdlib definition uri, got %s", response.Result.URI)
	}
	definitionPath := uriToPath(response.Result.URI)
	contents, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatalf("expected generated stdlib definition file: %v", err)
	}
	if !strings.Contains(string(contents), "array_copy( array )") {
		t.Fatalf("expected declaration signature to be present, got %q", string(contents))
	}
}

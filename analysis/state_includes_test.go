package analysis

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
)

func requireGscp(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gscp"); err != nil {
		t.Fatalf("gscp is required for tests: %v", err)
	}
}

func TestStdlibBundleHasCommonUtility(t *testing.T) {
	stdlib, err := StdlibSignatures()
	if err != nil {
		t.Fatalf("failed to load stdlib signatures: %v", err)
	}
	mp, ok := stdlib["mp"]
	if !ok {
		t.Fatalf("missing mp stdlib bundle")
	}
	utility, ok := mp["common_scripts/utility"]
	if !ok {
		t.Fatalf("missing common_scripts/utility in mp bundle")
	}
	if !hasFunction(utility, "array_copy") {
		t.Fatalf("common_scripts/utility missing array_copy signature")
	}
}

func TestStdlibBundleHasZmUtility(t *testing.T) {
	stdlib, err := StdlibSignatures()
	if err != nil {
		t.Fatalf("failed to load stdlib signatures: %v", err)
	}
	zm, ok := stdlib["zm"]
	if !ok {
		t.Fatalf("missing zm stdlib bundle")
	}
	utility, ok := zm["maps/mp/zombies/_zm_utility"]
	if !ok {
		t.Fatalf("missing maps/mp/zombies/_zm_utility in zm bundle")
	}
	if !hasFunction(utility, "init_utility") {
		t.Fatalf("_zm_utility missing init_utility signature")
	}
}

func TestHoverUsesIncludedStdlibMP(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "#include common_scripts\\utility;\n" +
		"main() { array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 1, "array_copy")
	response := state.Hover(1, uri, position)
	if !strings.Contains(response.Result.Contents, "array_copy (") {
		t.Fatalf("hover missing array_copy signature: %q", response.Result.Contents)
	}
}

func TestInlayHintsUseIncludedStdlibMP(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "#include common_scripts\\utility;\n" +
		"main() { array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "array: ") {
		t.Fatalf("missing inlay hint for array_copy: %v", response.Result)
	}
}

func TestHoverUsesIncludedStdlibZM(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/zm/maps/mp/zombies/test.gsc"
	text := "#include maps\\mp\\zombies\\_zm_utility;\n" +
		"main() { init_utility(); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 1, "init_utility")
	response := state.Hover(1, uri, position)
	if !strings.Contains(response.Result.Contents, "init_utility (") {
		t.Fatalf("hover missing init_utility signature: %q", response.Result.Contents)
	}
}

func TestInlayHintsUseIncludedStdlibZM(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/zm/maps/mp/zombies/test.gsc"
	text := "#include maps\\mp\\zombies\\_zm_utility;\n" +
		"main() { convertsecondstomilliseconds(1); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "seconds: ") {
		t.Fatalf("missing inlay hint for convertsecondstomilliseconds: %v", response.Result)
	}
}

func TestInlayHintsUseIncludedStdlibZMCaseInsensitive(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/zm/maps/mp/zombies/test.gsc"
	text := "#include maps\\mp\\gametypes_zm\\_hud_util;\n" +
		"main() { createFontString(\"objective\", 1.5); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "font: ") {
		t.Fatalf("missing inlay hint for createFontString: %v", response.Result)
	}
}

func TestInlayHintsUseBuiltinWithoutIncludes(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { waittill(\"ready\", player, data); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "event: ") {
		t.Fatalf("missing builtin waittill inlay hint: %v", response.Result)
	}
}

func TestBuiltinDoesNotOverrideLocalDeclaration(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "wait( duration ) { }\n" +
		"main() { wait(1); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "duration: ") {
		t.Fatalf("missing local wait inlay hint: %v", response.Result)
	}
}

func TestHoverUsesQualifiedStdlibMP(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { common_scripts\\utility::array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 0, "array_copy")
	response := state.Hover(1, uri, position)
	if !strings.Contains(response.Result.Contents, "array_copy (") {
		t.Fatalf("hover missing array_copy signature: %q", response.Result.Contents)
	}
}

func TestInlayHintsUseQualifiedStdlibMP(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { common_scripts\\utility::array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "array: ") {
		t.Fatalf("missing inlay hint for array_copy: %v", response.Result)
	}
}

func TestHoverUsesQualifiedStdlibZM(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/zm/maps/mp/zombies/test.gsc"
	text := "main() { maps\\mp\\zombies\\_zm_utility::init_utility(); }\n"

	state.OpenDocument(uri, text)
	position := positionForLine(text, 0, "init_utility")
	response := state.Hover(1, uri, position)
	if !strings.Contains(response.Result.Contents, "init_utility (") {
		t.Fatalf("hover missing init_utility signature: %q", response.Result.Contents)
	}
}

func TestInlayHintsUseQualifiedStdlibZM(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/zm/maps/mp/zombies/test.gsc"
	text := "main() { maps\\mp\\zombies\\_zm_utility::convertsecondstomilliseconds(1); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "seconds: ") {
		t.Fatalf("missing inlay hint for convertsecondstomilliseconds: %v", response.Result)
	}
}

func hasFunction(signatures []FunctionSignature, name string) bool {
	for _, sig := range signatures {
		if sig.Name == name {
			return true
		}
	}
	return false
}

func hasInlayLabel(hints []lsp.InlayHint, label string) bool {
	for _, hint := range hints {
		if hint.Label == label {
			return true
		}
	}
	return false
}

func positionForLine(text string, line int, needle string) lsp.Position {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return lsp.Position{Line: line, Character: 0}
	}
	col := strings.Index(lines[line], needle)
	if col < 0 {
		col = 0
	}
	return lsp.Position{Line: line, Character: col}
}

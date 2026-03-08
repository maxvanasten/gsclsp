package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maxvanasten/gsclsp/lsp"
)

func requireGscp(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gscp"); err != nil {
		t.Fatalf("gscp is required for tests: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func uriForPath(path string) string {
	path = filepath.ToSlash(path)
	return "file://" + path
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
	if !hasInlayLabel(response.Result, "common_scripts\\utility::") {
		t.Fatalf("missing include origin inlay hint for stdlib include: %v", response.Result)
	}
}

func TestInlayHintsUseIncludedLocalFile(t *testing.T) {
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
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "foo: ") {
		t.Fatalf("missing inlay hint for local helpers: %v", response.Result)
	}
	if !hasInlayLabel(response.Result, "helpers::") {
		t.Fatalf("missing include origin inlay hint for local helpers: %v", response.Result)
	}
}

func TestInlayHintsUseIncludedLocalNestedFile(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "test.gsc")
	helperPath := filepath.Join(dir, "common", "helpers.gsc")

	writeFile(t, helperPath, "nested_helper( value ) { }\n")
	text := "#include common\\helpers;\n" +
		"main() { nested_helper(thing); }\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "value: ") {
		t.Fatalf("missing inlay hint for nested local helpers: %v", response.Result)
	}
}

func TestInlayHintsUseIncludedLocalOpenCall(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "test.gsc")
	helperPath := filepath.Join(dir, "helpers.gsc")

	writeFile(t, helperPath, "helpers( name ) { }\n")
	text := "#include helpers;\n" +
		"main() { helpers(\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "name: ") {
		t.Fatalf("missing inlay hint for open local call: %v", response.Result)
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
	if !hasInlayLabel(response.Result, "maps\\mp\\zombies\\_zm_utility::") {
		t.Fatalf("missing include origin inlay hint for stdlib utility: %v", response.Result)
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

func TestInlayHintOriginForMethodCallAnchorsBeforeFunctionName(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/zm/maps/mp/zombies/test.gsc"
	text := "#include maps\\mp\\gametypes_zm\\_hud_util;\n" +
		"main() { self.gpp_ui_gg_hud setPoint(); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	originLabel := "maps\\mp\\gametypes_zm\\_hud_util::"
	hint, ok := findInlayHintByLabel(response.Result, originLabel)
	if !ok {
		t.Fatalf("missing include origin inlay hint for method call: %v", response.Result)
	}

	line := strings.Split(text, "\n")[1]
	functionCol := strings.Index(line, "setPoint")
	if functionCol < 0 {
		t.Fatalf("test setup missing function call in line: %q", line)
	}

	if hint.Position.Line != 1 {
		t.Fatalf("origin hint line = %d, want 1", hint.Position.Line)
	}
	if hint.Position.Character != functionCol {
		t.Fatalf("origin hint column = %d, want %d", hint.Position.Character, functionCol)
	}
}

func TestInlayHintOriginForThreadCallAnchorsBeforeFunctionName(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "#include common_scripts\\utility;\n" +
		"main() { level thread array_copy(foo); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	originLabel := "common_scripts\\utility::"
	hint, ok := findInlayHintByLabel(response.Result, originLabel)
	if !ok {
		t.Fatalf("missing include origin inlay hint for thread call: %v", response.Result)
	}

	line := strings.Split(text, "\n")[1]
	functionCol := strings.Index(line, "array_copy")
	if functionCol < 0 {
		t.Fatalf("test setup missing function call in line: %q", line)
	}

	if hint.Position.Line != 1 {
		t.Fatalf("origin hint line = %d, want 1", hint.Position.Line)
	}
	if hint.Position.Character != functionCol {
		t.Fatalf("origin hint column = %d, want %d", hint.Position.Character, functionCol)
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
	if hasAnyOriginHint(response.Result) {
		t.Fatalf("unexpected include origin inlay hint for builtin call: %v", response.Result)
	}
}

func TestInlayHintsUseBuiltinMethodWithoutIncludes(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "main() { self.gpp_ui_health_counter setValue(0); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "value: ") {
		t.Fatalf("missing builtin setValue inlay hint: %v", response.Result)
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
	if hasAnyOriginHint(response.Result) {
		t.Fatalf("unexpected include origin inlay hint for local declaration: %v", response.Result)
	}
}

func TestInlayHintsShowSelfContextForUniqueThreadReceiver(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "somefunc() { self iprintln(); }\n" +
		"main() { player thread somefunc(); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	if countInlayLabels(response.Result, " -> player") != 1 {
		t.Fatalf("expected one self-context inlay hint for player receiver, got: %v", response.Result)
	}

	hint, ok := findInlayHintByLabel(response.Result, " -> player")
	if !ok {
		t.Fatalf("missing self-context inlay hint for player receiver: %v", response.Result)
	}

	selfPos := positionForLine(text, 0, "self")
	selfEndPos := lsp.Position{Line: selfPos.Line, Character: selfPos.Character + len("self")}
	if hint.Position.Line != selfEndPos.Line || hint.Position.Character != selfEndPos.Character {
		t.Fatalf("self-context hint position = (%d,%d), want (%d,%d)", hint.Position.Line, hint.Position.Character, selfEndPos.Line, selfEndPos.Character)
	}
}

func TestInlayHintsDoNotShowSelfContextForAmbiguousThreadReceivers(t *testing.T) {
	requireGscp(t)
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"
	text := "somefunc() { self iprintln(); }\n" +
		"main() { player thread somefunc(); level thread somefunc(); }\n"

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	if hasInlayLabel(response.Result, " -> player") || hasInlayLabel(response.Result, " -> level") {
		t.Fatalf("expected no self-context inlay hint for ambiguous receivers, got: %v", response.Result)
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
	if hasInlayLabel(response.Result, "common_scripts\\utility::") {
		t.Fatalf("unexpected include origin inlay hint for qualified call: %v", response.Result)
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

func TestIncludeCacheInvalidatesOnFileChange(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "test.gsc")
	helperPath := filepath.Join(dir, "helpers.gsc")

	mainText := "#include helpers;\n" +
		"main() { helpers(1); }\n"
	writeFile(t, helperPath, "helpers( foo ) { }\n")
	writeFile(t, mainPath, mainText)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, mainText)
	if !hasInlayLabel(state.InlayHints(1, uri).Result, "foo: ") {
		t.Fatal("expected initial include signature to contain foo")
	}

	time.Sleep(20 * time.Millisecond)
	writeFile(t, helperPath, "helpers( changed_name ) { }\n")
	state.UpdateDocument(uri, mainText+" ")

	if !hasInlayLabel(state.InlayHints(1, uri).Result, "changed_name: ") {
		t.Fatal("expected include cache to invalidate after helper file change")
	}
}

func TestIncludeCacheReusesUnchangedFile(t *testing.T) {
	requireGscp(t)
	state := NewState()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "test.gsc")
	helperPath := filepath.Join(dir, "helpers.gsc")

	mainText := "#include helpers;\n" +
		"main() { helpers(1); }\n"
	writeFile(t, helperPath, "helpers( foo ) { }\n")
	writeFile(t, mainPath, mainText)

	uri := uriForPath(mainPath)
	state.OpenDocument(uri, mainText)
	if !hasInlayLabel(state.InlayHints(1, uri).Result, "foo: ") {
		t.Fatal("expected include signature to contain foo")
	}

	resolvedPath, ok := resolveIncludePath(uri, "helpers")
	if !ok {
		t.Fatal("expected include path to resolve")
	}
	resolvedPath = filepath.Clean(resolvedPath)
	if _, ok := state.includeCache[resolvedPath]; !ok {
		t.Fatal("expected include to be cached after initial parse")
	}

	for i := 0; i < 3; i++ {
		state.UpdateDocument(uri, mainText)
	}

	if len(state.includeCache) != 1 {
		t.Fatalf("expected one cache entry, got %d", len(state.includeCache))
	}
	if !hasInlayLabel(state.InlayHints(1, uri).Result, "foo: ") {
		t.Fatal("expected unchanged include cache to keep foo signature")
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

func findInlayHintByLabel(hints []lsp.InlayHint, label string) (lsp.InlayHint, bool) {
	for _, hint := range hints {
		if hint.Label == label {
			return hint, true
		}
	}
	return lsp.InlayHint{}, false
}

func hasAnyOriginHint(hints []lsp.InlayHint) bool {
	for _, hint := range hints {
		if strings.HasSuffix(hint.Label, "::") {
			return true
		}
	}
	return false
}

func countInlayLabels(hints []lsp.InlayHint, label string) int {
	count := 0
	for _, hint := range hints {
		if hint.Label == label {
			count++
		}
	}
	return count
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

func TestNormalizeIncludePathBase(t *testing.T) {
	cases := map[string]string{
		" common_scripts\\utility.gsc ": "common_scripts/utility",
		"/maps/mp/zombies/_zm_utility":  "maps/mp/zombies/_zm_utility",
		"./helpers":                     "./helpers",
		".\\helpers":                    "./helpers",
	}
	for input, expected := range cases {
		if got := normalizeIncludePathBase(input); got != expected {
			t.Fatalf("normalizeIncludePathBase(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestResolveIncludePathRelativePrefixes(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "test.gsc")
	helperPath := filepath.Join(root, "helpers.gsc")

	writeFile(t, mainPath, "main() { helpers(); }\n")
	writeFile(t, helperPath, "helpers() { }\n")

	uri := uriForPath(mainPath)
	for _, includePath := range []string{"./helpers", ".\\helpers"} {
		resolved, ok := resolveIncludePath(uri, includePath)
		if !ok {
			t.Fatalf("resolveIncludePath(%q) failed", includePath)
		}
		if filepath.Clean(resolved) != filepath.Clean(helperPath) {
			t.Fatalf("resolveIncludePath(%q) = %q, want %q", includePath, resolved, helperPath)
		}
	}
}

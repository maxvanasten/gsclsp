package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveIncludePathFromWorkspaceFolder(t *testing.T) {
	requireGscp(t)
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", scriptsDir, err)
	}

	mainPath := filepath.Join(root, "scripts", "zm", "test.gsc")
	helperPath := filepath.Join(root, "scripts", "zm", "maxlib.gsc")

	writeFile(t, mainPath, "main() { my_func(); }\n")
	writeFile(t, helperPath, "my_func( value ) { }\n")

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root

	resolved, ok := resolveIncludePath(uri, "scripts\\zm\\maxlib", []string{workspaceRoot})
	if !ok {
		t.Fatalf("resolveIncludePath with workspace folder failed")
	}
	if filepath.Clean(resolved) != filepath.Clean(helperPath) {
		t.Fatalf("resolveIncludePath(%q) = %q, want %q", "scripts\\zm\\maxlib", resolved, helperPath)
	}
}

func TestDetectWorkspaceRootFromDocument(t *testing.T) {
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", scriptsDir, err)
	}

	docPath := filepath.Join(root, "scripts", "zm", "test.gsc")
	writeFile(t, docPath, "main() { }\n")

	uri := uriForPath(docPath)
	detected := DetectWorkspaceRootFromDocument(uri)

	if detected != root {
		t.Fatalf("DetectWorkspaceRootFromDocument(%q) = %q, want %q", uri, detected, root)
	}
}

func TestDetectWorkspaceRootFromDocumentNoScripts(t *testing.T) {
	root := t.TempDir()
	docPath := filepath.Join(root, "test.gsc")
	writeFile(t, docPath, "main() { }\n")

	uri := uriForPath(docPath)
	detected := DetectWorkspaceRootFromDocument(uri)

	if detected != "" {
		t.Fatalf("DetectWorkspaceRootFromDocument(%q) = %q, want empty", uri, detected)
	}
}

func TestInlayHintsUseIncludedLocalFileFromWorkspaceFolder(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", scriptsDir, err)
	}

	mainPath := filepath.Join(scriptsDir, "test.gsc")
	helperPath := filepath.Join(scriptsDir, "maxlib.gsc")

	writeFile(t, helperPath, "maxlib_helper( foo ) { }\n")
	text := "#include scripts\\zm\\maxlib;\n" +
		"main() { maxlib_helper(bar); }\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})

	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	if !hasInlayLabel(response.Result, "foo: ") {
		t.Fatalf("missing inlay hint for workspace folder helpers: %v", response.Result)
	}
	if !hasInlayLabel(response.Result, "scripts\\zm\\maxlib::") {
		t.Fatalf("missing include origin inlay hint for workspace folder helpers: %v", response.Result)
	}
}

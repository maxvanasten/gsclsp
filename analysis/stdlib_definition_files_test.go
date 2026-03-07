package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderStdlibDefinitionFileUsesDeclarationBody(t *testing.T) {
	entries := []StdlibDeclaration{{
		Name:        "array_copy",
		Arguments:   []string{"array"},
		Declaration: "array_copy( array ) {\n\treturn array;\n}",
	}}

	content, ranges := renderStdlibDefinitionFile(entries)
	if content == "" {
		t.Fatal("expected rendered content")
	}
	if !strings.Contains(content, "return array;") {
		t.Fatalf("expected declaration body in output, got %q", content)
	}
	r, ok := ranges["array_copy"]
	if !ok {
		t.Fatal("expected function range for array_copy")
	}
	if r.Start.Line != 0 || r.Start.Character != 0 {
		t.Fatalf("unexpected range start: %+v", r.Start)
	}
}

func TestMergeDeclarationEntriesAddsFallbackForMissingDeclaration(t *testing.T) {
	entries := mergeDeclarationEntries(
		[]StdlibDeclaration{{Name: "real_fn", Arguments: []string{"x"}, Declaration: "real_fn( x ) {\n\treturn x;\n}"}},
		[]FunctionSignature{{Name: "real_fn", Arguments: []string{"x"}}, {Name: "missing_fn", Arguments: []string{"a", "b"}}},
	)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "missing_fn" || entries[1].Name != "real_fn" {
		t.Fatalf("unexpected sorted entries: %+v", entries)
	}
	if !strings.Contains(entries[0].Declaration, "missing_fn(a, b)") {
		t.Fatalf("expected fallback declaration for missing_fn, got %q", entries[0].Declaration)
	}
}

func TestPruneStdlibDefinitionRootsSkipsActivePID(t *testing.T) {
	tempDir := t.TempDir()
	staleRoot := filepath.Join(tempDir, stdlibDefinitionRootPrefix+"101-a")
	activeRoot := filepath.Join(tempDir, stdlibDefinitionRootPrefix+"202-b")
	legacyRoot := filepath.Join(tempDir, stdlibDefinitionRootPrefix+"legacy")
	unrelatedRoot := filepath.Join(tempDir, "not-gsclsp")

	for _, dir := range []string{staleRoot, activeRoot, legacyRoot, unrelatedRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	err := pruneStdlibDefinitionRoots(tempDir, func(pid int) bool { return pid == 202 })
	if err != nil {
		t.Fatalf("prune roots: %v", err)
	}

	if _, err := os.Stat(staleRoot); !os.IsNotExist(err) {
		t.Fatalf("expected stale root removed, stat err: %v", err)
	}
	if _, err := os.Stat(activeRoot); err != nil {
		t.Fatalf("expected active root kept: %v", err)
	}
	if _, err := os.Stat(legacyRoot); !os.IsNotExist(err) {
		t.Fatalf("expected legacy root removed, stat err: %v", err)
	}
	if _, err := os.Stat(unrelatedRoot); err != nil {
		t.Fatalf("expected unrelated root kept: %v", err)
	}
}

func TestStateCloseRemovesStdlibDefinitionRoot(t *testing.T) {
	state := NewState()
	root := filepath.Join(t.TempDir(), stdlibDefinitionRootPrefix+"777-x")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	state.stdlibDefinitionRoot = root

	if err := state.Close(); err != nil {
		t.Fatalf("close state: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected stdlib definition root removed, stat err: %v", err)
	}
}

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maxvanasten/gsclsp/analysis"
	p "github.com/maxvanasten/gscp/parser"
)

func TestNormalizeKeyWithoutPrefix(t *testing.T) {
	got := normalizeKey("Common_Scripts\\Utility.gsc", "")
	if got != "common_scripts/utility" {
		t.Fatalf("unexpected key: %q", got)
	}
}

func TestNormalizeKeyWithPrefix(t *testing.T) {
	got := normalizeKey("Zm_Tomb.GSC", "maps/mp")
	if got != "maps/mp/zm_tomb" {
		t.Fatalf("unexpected key: %q", got)
	}
}

func TestMergeSignaturesDeduplicates(t *testing.T) {
	base := []analysis.FunctionSignature{{Name: "foo", Arguments: []string{"a"}}}
	add := []analysis.FunctionSignature{
		{Name: "foo", Arguments: []string{"a"}},
		{Name: "foo", Arguments: []string{"b"}},
	}

	merged := mergeSignatures(base, add)
	if len(merged) != 2 {
		t.Fatalf("expected 2 signatures, got %d", len(merged))
	}
}

func TestMergeSignatureMapsMergesSharedKeys(t *testing.T) {
	base := map[string][]analysis.FunctionSignature{
		"maps/mp/zm_tomb": {{Name: "foo", Arguments: []string{"a"}}},
	}
	add := map[string][]analysis.FunctionSignature{
		"maps/mp/zm_tomb":   {{Name: "foo", Arguments: []string{"a"}}, {Name: "bar", Arguments: nil}},
		"maps/mp/zm_prison": {{Name: "baz", Arguments: nil}},
	}

	mergeSignatureMaps(base, add)

	if len(base) != 2 {
		t.Fatalf("expected 2 map keys, got %d", len(base))
	}

	wantTomb := []analysis.FunctionSignature{{Name: "foo", Arguments: []string{"a"}}, {Name: "bar", Arguments: nil}}
	if !reflect.DeepEqual(base["maps/mp/zm_tomb"], wantTomb) {
		t.Fatalf("unexpected merged signatures: %#v", base["maps/mp/zm_tomb"])
	}
}

func TestBuildGroupSignatureMapSkipsEmptyRoots(t *testing.T) {
	m, declarations, duplicates, err := buildGroupSignatureMap([]signatureRoot{{Path: ""}, {Path: "   "}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d keys", len(m))
	}
	if len(declarations) != 0 {
		t.Fatalf("expected empty declarations, got %d keys", len(declarations))
	}
	if len(duplicates) != 0 {
		t.Fatalf("expected no duplicates, got %d", len(duplicates))
	}
}

func TestSliceNodeText(t *testing.T) {
	source := "alpha() {\n  beta();\n}\n"
	node := p.Node{Line: 1, Col: 1, Length: len("alpha() {\n  beta();\n}")}
	got := sliceNodeText(source, node)
	if got != "alpha() {\n  beta();\n}" {
		t.Fatalf("unexpected node slice: %q", got)
	}
}

func TestFindMapRuntimeRoot(t *testing.T) {
	root := t.TempDir()
	mapDir := filepath.Join(root, "zm_tomb")
	runtimeDir := filepath.Join(mapDir, "maps", "mp")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}

	got, ok := findMapRuntimeRoot(mapDir)
	if !ok {
		t.Fatal("expected runtime root to be found")
	}
	if got != runtimeDir {
		t.Fatalf("unexpected runtime root: %q", got)
	}
}

func TestFindMapRuntimeRootMissing(t *testing.T) {
	root := t.TempDir()
	mapDir := filepath.Join(root, "missing")
	if err := os.MkdirAll(mapDir, 0o755); err != nil {
		t.Fatalf("mkdir map dir: %v", err)
	}

	_, ok := findMapRuntimeRoot(mapDir)
	if ok {
		t.Fatal("expected runtime root to be missing")
	}
}

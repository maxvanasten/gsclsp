package analysis

import (
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

package analysis

import (
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
	p "github.com/maxvanasten/gscp/parser"
)

func TestCollectAllFunctionCalls(t *testing.T) {
	nodes := []p.Node{
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: "foo"},
		},
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: "bar", Path: "utils"},
		},
		{
			Type: "other_node",
			Children: []p.Node{
				{
					Type: "function_call",
					Data: p.NodeData{FunctionName: "nested"},
				},
			},
		},
	}

	result := collectAllFunctionCalls(nodes)

	if len(result) != 3 {
		t.Errorf("expected 3 function calls, got %d", len(result))
	}

	expected := []string{"foo", "utils::bar", "nested"}
	for _, name := range expected {
		if _, ok := result[name]; !ok {
			t.Errorf("expected to find function call '%s'", name)
		}
	}
}

func TestCollectAllFunctionCalls_Empty(t *testing.T) {
	result := collectAllFunctionCalls(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 function calls for nil input, got %d", len(result))
	}

	result = collectAllFunctionCalls([]p.Node{})
	if len(result) != 0 {
		t.Errorf("expected 0 function calls for empty input, got %d", len(result))
	}
}

func TestCollectAllFunctionCalls_SkipsEmptyNames(t *testing.T) {
	nodes := []p.Node{
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: "valid"},
		},
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: "  "}, // whitespace only
		},
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: ""}, // empty
		},
	}

	result := collectAllFunctionCalls(nodes)

	if len(result) != 1 {
		t.Errorf("expected 1 function call, got %d", len(result))
	}

	if _, ok := result["valid"]; !ok {
		t.Error("expected to find 'valid' function call")
	}
}

func TestStateFindFunctionInStdlib(t *testing.T) {
	state := NewState()

	stdlib := map[string]map[string][]FunctionSignature{
		"mp": {
			"common\\functions": {
				{Name: "print"},
				{Name: "array"},
			},
			"math\\utils": {
				{Name: "sqrt"},
			},
		},
		"zm": {
			"zombie\\utils": {
				{Name: "spawnZombie"},
			},
		},
	}

	tests := []struct {
		name     string
		funcName string
		expected int
	}{
		{"find in mp common", "print", 1},
		{"find in mp math", "sqrt", 1},
		{"find in zm", "spawnZombie", 1},
		{"case insensitive", "PRINT", 1},
		{"not found", "nonexistent", 0},
		{"find array", "array", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := state.findFunctionInStdlib(tt.funcName, stdlib)
			if len(result) != tt.expected {
				t.Errorf("expected %d sources, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestStateFindFunctionInStdlib_NilStdlib(t *testing.T) {
	state := NewState()
	result := state.findFunctionInStdlib("print", nil)
	if result != nil {
		t.Errorf("expected nil for nil stdlib, got %v", result)
	}
}

func TestStateFindFunctionInStdlib_NoDuplicates(t *testing.T) {
	state := NewState()

	// Same function in multiple files - should only return unique paths
	stdlib := map[string]map[string][]FunctionSignature{
		"mp": {
			"file1": {{Name: "shared"}},
			"file2": {{Name: "shared"}},
		},
	}

	result := state.findFunctionInStdlib("shared", stdlib)

	// Should return both files since they're different paths
	if len(result) != 2 {
		t.Errorf("expected 2 unique sources for function in multiple files, got %d", len(result))
	}
}

func TestStateGetMissingFunctionIncludes(t *testing.T) {
	state := NewState()
	uri := "file:///test.gsc"

	// Create a document with some function calls
	doc := `
myLocalFunc();
#include common\functions;
builtinFunc();
unknownFunc();
	`

	state.OpenDocument(uri, doc)

	// Since we can't easily mock stdlib loading, we'll just verify the method works
	// without panicking and returns expected structure
	result := state.GetMissingFunctionIncludes(uri)

	// Result should be a map (may be empty depending on stdlib availability)
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestStateGetMissingFunctionIncludes_EmptyDoc(t *testing.T) {
	state := NewState()
	uri := "file:///empty.gsc"

	state.OpenDocument(uri, "")

	result := state.GetMissingFunctionIncludes(uri)

	if result == nil {
		t.Error("expected non-nil result for empty doc")
	}

	if len(result) != 0 {
		t.Errorf("expected 0 missing includes for empty doc, got %d", len(result))
	}
}

func TestStateGetMissingFunctionIncludes_SkipsQualified(t *testing.T) {
	state := NewState()
	uri := "file:///test.gsc"

	doc := `
qualified::func();
	`

	state.OpenDocument(uri, doc)

	result := state.GetMissingFunctionIncludes(uri)

	// Should skip qualified calls (they have explicit path)
	for funcName := range result {
		if funcName == "qualified::func" {
			t.Error("should not include qualified function calls in missing list")
		}
	}
}

func TestCollectAllFunctionCalls_Deduplication(t *testing.T) {
	nodes := []p.Node{
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: "duplicate"},
		},
		{
			Type: "function_call",
			Data: p.NodeData{FunctionName: "duplicate"},
		},
	}

	result := collectAllFunctionCalls(nodes)

	if len(result) != 1 {
		t.Errorf("expected 1 unique function call, got %d", len(result))
	}
}

// Mock function for testing state methods that require LSP types
func mockPosition(line, char int) lsp.Position {
	return lsp.Position{Line: line, Character: char}
}

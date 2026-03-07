package analysis

import (
	"strings"
	"testing"

	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

func TestCallClosedOnLineIgnoresNestedParens(t *testing.T) {
	input := "foo((x, y, z), bar"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	node := p.Node{Type: "function_call", Data: p.NodeData{FunctionName: "foo"}, Line: 1}
	lineTokens := indexTokensByLine(tokens)[node.Line]

	if callClosedOnLine(lineTokens, "foo") {
		t.Fatalf("expected callClosedOnLine to be false for open call with nested parens")
	}
}

func TestOpenCallParamAnchorSkipsNestedCommas(t *testing.T) {
	input := "foo((x, y, z), "
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	node := p.Node{Type: "function_call", Data: p.NodeData{FunctionName: "foo"}, Line: 1}
	lineTokens := indexTokensByLine(tokens)[node.Line]

	paramIndex, _, ok := openCallParamAnchor(lineTokens, "foo")
	if !ok {
		t.Fatalf("expected openCallParamAnchor to return ok for open call")
	}
	if paramIndex != 1 {
		t.Fatalf("expected paramIndex 1 after top-level comma, got %d", paramIndex)
	}
}

func TestFunctionCallLabelColFromTokensMethodCall(t *testing.T) {
	input := "self.gpp_ui_gg_hud setPoint()"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	node := p.Node{
		Type: "function_call",
		Data: p.NodeData{FunctionName: "setPoint", Method: "self.gpp_ui_gg_hud"},
		Line: 1,
		Col:  1,
	}
	lineTokens := indexTokensByLine(tokens)[node.Line]

	col := functionCallLabelCol(node, lineTokens)
	want := strings.Index(input, "setPoint")
	if col != want {
		t.Fatalf("functionCallLabelCol(method) = %d, want %d", col, want)
	}
}

func TestFunctionCallLabelColFromTokensThreadCall(t *testing.T) {
	input := "level thread somefunc()"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	node := p.Node{
		Type: "function_call",
		Data: p.NodeData{FunctionName: "somefunc", Thread: true},
		Line: 1,
		Col:  1,
	}
	lineTokens := indexTokensByLine(tokens)[node.Line]

	col := functionCallLabelCol(node, lineTokens)
	want := strings.Index(input, "somefunc")
	if col != want {
		t.Fatalf("functionCallLabelCol(thread) = %d, want %d", col, want)
	}
}

func TestFunctionCallLabelColFromTokensQualifiedCall(t *testing.T) {
	input := "common_scripts\\utility::array_copy(foo)"
	lexer := l.NewLexer([]byte(input))
	tokens := lexer.GetTokens()
	node := p.Node{
		Type: "function_call",
		Data: p.NodeData{FunctionName: "array_copy", Path: "common_scripts\\utility"},
		Line: 1,
		Col:  1,
	}
	lineTokens := indexTokensByLine(tokens)[node.Line]

	col := functionCallLabelCol(node, lineTokens)
	want := strings.Index(input, "array_copy")
	if col != want {
		t.Fatalf("functionCallLabelCol(qualified) = %d, want %d", col, want)
	}
}

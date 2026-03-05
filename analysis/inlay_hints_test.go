package analysis

import (
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

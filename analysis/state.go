package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/maxvanasten/gsclsp/lsp"
	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

type ParseResult struct {
	Ast    []p.Node  `json:"ast"`
	Tokens []l.Token `json:"tokens"`
}

type State struct {
	Documents map[string]string
	Ast       map[string][]p.Node
	Tokens    map[string][]l.Token
}

func NewState() State {
	return State{Documents: map[string]string{}, Ast: map[string][]p.Node{}, Tokens: map[string][]l.Token{}}
}

func (s *State) OpenDocument(uri, text string) {
	s.Documents[uri] = text
	s.UpdateAst(uri)
}

func (s *State) UpdateDocument(uri, text string) {
	s.Documents[uri] = text
	s.UpdateAst(uri)
}

func (s *State) UpdateAst(uri string) {
	_, file_path, found := bytes.Cut([]byte(uri), []byte("file://"))
	if !found {
		fmt.Fprintln(os.Stderr, "FILE NOT FOUND")
		os.Exit(1)
	}
	cmd := exec.Command("gscp", "-p", string(file_path))
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR RUNNING CMD: %v\n", err)
		os.Exit(1)
	}

	var parse_result ParseResult
	if err := json.Unmarshal(output, &parse_result); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR UNMARSHALING JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "parse_result!!!!!!!!!!!: %v\n", parse_result.Ast)

	s.Ast[uri] = parse_result.Ast
	s.Tokens[uri] = parse_result.Tokens
}

func (s *State) Hover(id int, uri string, position lsp.Position) lsp.HoverResponse {
	// Analyze AST to find if token under cursor matches function signature

	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: "Hello, from lsp",
		},
	}
}

func (s *State) Definition(id int, uri string, position lsp.Position) lsp.DefinitionResponse {
	return lsp.DefinitionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.Location{
			URI: uri,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      position.Line - 1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      position.Line - 1,
					Character: 0,
				},
			},
		},
	}
}

func (s *State) SemanticTokens(id int, uri string) lsp.SemanticTokensResponse {
	tokens := GenerateSemanticTokens(s.Tokens[uri])

	return lsp.SemanticTokensResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.SemanticTokensResult{
			Data: tokens,
		},
	}
}

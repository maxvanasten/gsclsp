package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

type ParseResult struct {
	Ast    []p.Node  `json:"ast"`
	Tokens []l.Token `json:"tokens"`
}

type State struct {
	Documents  map[string]string
	Ast        map[string][]p.Node
	Tokens     map[string][]l.Token
	Signatures map[string][]FunctionSignature
}

func NewState() State {
	return State{Documents: map[string]string{}, Ast: map[string][]p.Node{}, Tokens: map[string][]l.Token{}, Signatures: map[string][]FunctionSignature{}}
}

func (s *State) OpenDocument(uri, text string) {
	s.Documents[uri] = text
	s.UpdateAst(uri)
}

func (s *State) UpdateDocument(uri, text string) {
	s.Documents[uri] = text
	s.UpdateAst(uri)
}

func Parse(input string) ParseResult {
	cmd := exec.Command("gscp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse() error while piping stdin: %v\n", err)
		os.Exit(1)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse() error while combining output: %v\n", err)
		os.Exit(1)
	}

	var parseResult ParseResult
	if err = json.Unmarshal(out, &parseResult); err != nil {
		fmt.Fprintf(os.Stderr, "Parse() error while unmarshaling json: %v\n", err)
		os.Exit(1)
	}

	return parseResult
}

// AddDocument Parses a file and adds all relevant nodes (function signatures) to the states document
func (s *State) AddDocument(uri, rootDir string, filePath string) {
	relPath := strings.Builder{}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR READING FILE (state.AddDocument): %v\n", err)
		os.Exit(1)
	}
	relPath.WriteString(wd)
	relPath.WriteString(rootDir)
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	relPath.WriteString(filePath)
	relPath.WriteString(".gsc")
	data, err := os.ReadFile(relPath.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR READING FILE (state.AddDocument): %v\n", err)
		os.Exit(1)
	}

	parseResult := Parse(string(data))

	s.Signatures[uri] = append(s.Signatures[uri], GenerateFunctionSignatures(parseResult.Ast)...)
}

func (s *State) UpdateAst(uri string) {
	parseResult := Parse(s.Documents[uri])

	s.Ast[uri] = parseResult.Ast
	s.Tokens[uri] = parseResult.Tokens
	s.Signatures[uri] = GenerateFunctionSignatures(s.Ast[uri])

	// TODO: Load included files
	// Convert filepath to actual location relative to gsclsp/lib
	// for _, n := range s.Ast[uri] {
	//		if n.Type == "include_statement" {
	//		s.AddDocument(uri, "/lib/zm/core/", n.Data.Path)
	//	}
	//}
}

func (s *State) GetTokenAtPosition(uri string, position lsp.Position) l.Token {
	for _, t := range s.Tokens[uri] {
		tokenLine := t.Line - 1
		tokenStartCol := t.Col - 1
		tokenEndCol := t.EndCol - 1
		if position.Line == tokenLine {
			if position.Character >= tokenStartCol && position.Character <= tokenEndCol {
				return t
			}
		}
	}

	return l.Token{}
}

func (s *State) Hover(id int, uri string, position lsp.Position) lsp.HoverResponse {
	output := strings.Builder{}

	token := s.GetTokenAtPosition(uri, position)
	fmt.Fprintf(os.Stderr, "HOVER TOKEN: %v\n", token)
	if token.Type == l.SYMBOL {
		fmt.Fprintf(os.Stderr, "signatures: \n%v\n", s.Signatures[uri])
		for _, s := range s.Signatures[uri] {
			if s.Name == token.Content {
				output.WriteString(s.Name)
				output.WriteString(" (")
				for i, a := range s.Arguments {
					output.WriteString(a)
					if i+1 < len(s.Arguments) {
						output.WriteString(", ")
					}
				}
				output.WriteString(")")
			}
		}
	}

	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: output.String(),
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

func (s *State) InlayHints(id int, uri string) lsp.InlayHintResponse {
	inlayHints := GenerateInlayHints(s.Signatures[uri], s.Ast[uri])

	fmt.Fprintf(os.Stderr, "inlayHints: %v\n", inlayHints)

	return lsp.InlayHintResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: inlayHints,
	}
}

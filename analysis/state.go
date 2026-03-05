package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
func (s *State) AddDocument(uri, filePath string) {
	resolvedPath, ok := resolveIncludePath(uri, filePath)
	if !ok {
		fmt.Fprintf(os.Stderr, "ERROR RESOLVING INCLUDE (state.AddDocument): %s\n", filePath)
		return
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR READING FILE (state.AddDocument): %v\n", err)
		return
	}

	parseResult := Parse(string(data))

	s.Signatures[uri] = mergeSignatures(s.Signatures[uri], GenerateFunctionSignatures(parseResult.Ast))
}

func resolveIncludePath(uri, includePath string) (string, bool) {
	includePath = strings.ReplaceAll(includePath, "\\", "/")
	includePath = strings.TrimSpace(includePath)
	includePath = strings.TrimSuffix(includePath, ".gsc")
	includePath = strings.TrimPrefix(includePath, "/")
	includePath = strings.TrimPrefix(includePath, "./")
	includePath = strings.TrimPrefix(includePath, ".\\")
	if includePath == "" {
		return "", false
	}

	relativePath := filepath.FromSlash(includePath + ".gsc")
	if filepath.IsAbs(relativePath) {
		if _, err := os.Stat(relativePath); err == nil {
			return relativePath, true
		}
		return "", false
	}

	if docPath := uriToPath(uri); docPath != "" {
		candidate := filepath.Join(filepath.Dir(docPath), relativePath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}

	return "", false
}

func uriToPath(uri string) string {
	if uri == "" {
		return ""
	}
	if strings.HasPrefix(uri, "file://") {
		parsed, err := url.Parse(uri)
		if err == nil {
			return filepath.FromSlash(parsed.Path)
		}
		return filepath.FromSlash(strings.TrimPrefix(uri, "file://"))
	}
	return uri
}

func (s *State) UpdateAst(uri string) {
	parseResult := Parse(s.Documents[uri])

	s.Ast[uri] = parseResult.Ast
	s.Tokens[uri] = parseResult.Tokens
	s.Signatures[uri] = GenerateFunctionSignatures(s.Ast[uri])

	builtins, err := BuiltinsSignatures()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR LOADING BUILTIN SIGNATURES: %v\n", err)
	} else {
		s.Signatures[uri] = mergeSignatures(s.Signatures[uri], builtins)
	}

	stdlib, err := StdlibSignatures()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR LOADING STDLIB SIGNATURES: %v\n", err)
	}

	includePaths := collectIncludePaths(s.Ast[uri])
	stdlibGroup := guessStdlibGroup(uri)

	for _, includePath := range includePaths {
		key := normalizeIncludeKey(includePath)
		if key == "" {
			continue
		}

		var matched bool
		if stdlib != nil {
			if stdlibGroup != "" {
				if sigs, ok := stdlib[stdlibGroup][key]; ok {
					s.Signatures[uri] = mergeSignatures(s.Signatures[uri], sigs)
					matched = true
				}
			}
			if !matched {
				if sigs, ok := stdlib["mp"][key]; ok {
					s.Signatures[uri] = mergeSignatures(s.Signatures[uri], sigs)
					matched = true
				}
			}
			if !matched {
				if sigs, ok := stdlib["zm"][key]; ok {
					s.Signatures[uri] = mergeSignatures(s.Signatures[uri], sigs)
					matched = true
				}
			}
		}

		if !matched {
			s.AddDocument(uri, includePath)
		}
	}
}

func collectIncludePaths(nodes []p.Node) []string {
	paths := []string{}
	for _, n := range nodes {
		if n.Type == "include_statement" && n.Data.Path != "" {
			paths = append(paths, n.Data.Path)
		}
		if len(n.Children) > 0 {
			paths = append(paths, collectIncludePaths(n.Children)...)
		}
	}

	return paths
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
		stdlib, _ := StdlibSignatures()
		name := token.Content
		sig, ok := resolveSignatureForName(name, uri, s.Signatures[uri], stdlib)
		if !ok {
			name = findFunctionCallNameAtPosition(s.Ast[uri], position)
			sig, ok = resolveSignatureForName(name, uri, s.Signatures[uri], stdlib)
		}
		if ok {
			output.WriteString(sig.Name)
			output.WriteString(" (")
			for i, a := range sig.Arguments {
				output.WriteString(a)
				if i+1 < len(sig.Arguments) {
					output.WriteString(", ")
				}
			}
			output.WriteString(")")
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
	stdlib, _ := StdlibSignatures()
	resolver := func(name string) (FunctionSignature, bool) {
		return resolveSignatureForName(name, uri, s.Signatures[uri], stdlib)
	}
	inlayHints := GenerateInlayHints(s.Signatures[uri], s.Ast[uri], s.Tokens[uri], resolver)

	fmt.Fprintf(os.Stderr, "inlayHints: %v\n", inlayHints)

	return lsp.InlayHintResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: inlayHints,
	}
}

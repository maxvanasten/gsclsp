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
	"github.com/maxvanasten/gscp/diagnostics"
	l "github.com/maxvanasten/gscp/lexer"
	p "github.com/maxvanasten/gscp/parser"
)

type ParseResult struct {
	Ast         []p.Node                 `json:"ast"`
	Tokens      []l.Token                `json:"tokens"`
	Diagnostics []diagnostics.Diagnostic `json:"diagnostics"`
}

type State struct {
	Documents      map[string]string
	Ast            map[string][]p.Node
	Tokens         map[string][]l.Token
	Signatures     map[string][]FunctionSignature
	Diagnostics    map[string][]lsp.Diagnostic
	includeCache   map[string]includeCacheEntry
	stdlib         map[string]map[string][]FunctionSignature
	builtins       []FunctionSignature
	stdlibErr      error
	builtinsErr    error
	stdlibLoaded   bool
	builtinsLoaded bool
}

type includeCacheEntry struct {
	ModTimeUnixNano int64
	Size            int64
	Ast             []p.Node
	Signatures      []FunctionSignature
}

func NewState() State {
	return State{
		Documents:    map[string]string{},
		Ast:          map[string][]p.Node{},
		Tokens:       map[string][]l.Token{},
		Signatures:   map[string][]FunctionSignature{},
		Diagnostics:  map[string][]lsp.Diagnostic{},
		includeCache: map[string]includeCacheEntry{},
	}
}

func (s *State) OpenDocument(uri, text string) {
	s.Documents[uri] = text
	s.UpdateAst(uri)
}

func (s *State) UpdateDocument(uri, text string) {
	if existing, ok := s.Documents[uri]; ok && existing == text {
		return
	}
	s.Documents[uri] = text
	s.UpdateAst(uri)
}

func Parse(input string) (ParseResult, error) {
	cmd := exec.Command("gscp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return ParseResult{}, fmt.Errorf("parse stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ParseResult{}, fmt.Errorf("gscp execution failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	var parseResult ParseResult
	if err = json.Unmarshal(out, &parseResult); err != nil {
		return ParseResult{}, fmt.Errorf("parse output json: %w", err)
	}

	return parseResult, nil
}

// AddDocument Parses a file and adds all relevant nodes (function signatures) to the states document
func (s *State) AddDocument(uri, filePath string) error {
	resolvedPath, ok := resolveIncludePath(uri, filePath)
	if !ok {
		return fmt.Errorf("resolve include path: %s", filePath)
	}

	entry, err := s.getParsedInclude(resolvedPath)
	if err != nil {
		return fmt.Errorf("parse include file %q: %w", resolvedPath, err)
	}
	s.Signatures[uri] = mergeSignatures(s.Signatures[uri], entry.Signatures)
	return nil
}

func (s *State) getParsedInclude(path string) (includeCacheEntry, error) {
	path = filepath.Clean(path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return includeCacheEntry{}, fmt.Errorf("stat include file: %w", err)
	}

	if cached, ok := s.includeCache[path]; ok {
		if cached.ModTimeUnixNano == fileInfo.ModTime().UnixNano() && cached.Size == fileInfo.Size() {
			return cached, nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return includeCacheEntry{}, fmt.Errorf("read include file: %w", err)
	}
	parseResult, err := Parse(string(data))
	if err != nil {
		return includeCacheEntry{}, err
	}

	entry := includeCacheEntry{
		ModTimeUnixNano: fileInfo.ModTime().UnixNano(),
		Size:            fileInfo.Size(),
		Ast:             parseResult.Ast,
		Signatures:      GenerateFunctionSignatures(parseResult.Ast),
	}
	s.includeCache[path] = entry
	return entry, nil
}

func resolveIncludePath(uri, includePath string) (string, bool) {
	includePath = normalizeIncludePathBase(includePath)
	includePath = strings.TrimPrefix(includePath, "./")
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
	if err := s.parseAndStore(uri); err != nil {
		s.Diagnostics[uri] = []lsp.Diagnostic{parseFailureDiagnostic(err)}
		return
	}
	s.mergeBuiltins(uri)
	stdlib := s.loadStdlib()
	includePaths := collectIncludePaths(s.Ast[uri])
	stdlibGroup := guessStdlibGroup(uri)
	s.applyIncludes(uri, includePaths, stdlibGroup, stdlib)
}

func (s *State) parseAndStore(uri string) error {
	parseResult, err := Parse(s.Documents[uri])
	if err != nil {
		return err
	}
	s.Ast[uri] = parseResult.Ast
	s.Tokens[uri] = parseResult.Tokens
	s.Signatures[uri] = GenerateFunctionSignatures(s.Ast[uri])
	s.Diagnostics[uri] = toLspDiagnostics(parseResult.Diagnostics)
	return nil
}

func (s *State) mergeBuiltins(uri string) {
	builtins, err := s.loadBuiltins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR LOADING BUILTIN SIGNATURES: %v\n", err)
		return
	}
	s.Signatures[uri] = mergeSignatures(s.Signatures[uri], builtins)
}

func (s *State) loadBuiltins() ([]FunctionSignature, error) {
	if s.builtinsLoaded {
		return s.builtins, s.builtinsErr
	}
	s.builtins, s.builtinsErr = BuiltinsSignatures()
	s.builtinsLoaded = true
	return s.builtins, s.builtinsErr
}

func (s *State) loadStdlib() map[string]map[string][]FunctionSignature {
	if s.stdlibLoaded {
		return s.stdlib
	}
	s.stdlib, s.stdlibErr = StdlibSignatures()
	s.stdlibLoaded = true
	if s.stdlibErr != nil {
		fmt.Fprintf(os.Stderr, "ERROR LOADING STDLIB SIGNATURES: %v\n", s.stdlibErr)
	}
	return s.stdlib
}

func (s *State) applyIncludes(uri string, includePaths []string, stdlibGroup string, stdlib map[string]map[string][]FunctionSignature) {
	for _, includePath := range includePaths {
		key := normalizeIncludeKey(includePath)
		if key == "" {
			continue
		}

		if sigs, ok := lookupStdlibSignatures(stdlib, stdlibGroup, key); ok {
			s.Signatures[uri] = mergeSignatures(s.Signatures[uri], sigs)
			continue
		}

		if err := s.AddDocument(uri, includePath); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR APPLYING INCLUDE %q: %v\n", includePath, err)
		}
	}
}

func lookupStdlibSignatures(stdlib map[string]map[string][]FunctionSignature, stdlibGroup, key string) ([]FunctionSignature, bool) {
	if stdlib == nil {
		return nil, false
	}
	if stdlibGroup != "" {
		if sigs, ok := stdlib[stdlibGroup][key]; ok {
			return sigs, true
		}
	}
	if sigs, ok := stdlib["mp"][key]; ok {
		return sigs, true
	}
	if sigs, ok := stdlib["zm"][key]; ok {
		return sigs, true
	}
	return nil, false
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
		if position.Line != t.Line-1 {
			continue
		}
		startCol := t.Col - 1
		endCol := t.EndCol - 1
		if position.Character < startCol || position.Character > endCol {
			continue
		}
		return t
	}

	return l.Token{}
}

func (s *State) Hover(id int, uri string, position lsp.Position) lsp.HoverResponse {
	output := strings.Builder{}

	token := s.GetTokenAtPosition(uri, position)
	if token.Type == l.SYMBOL {
		stdlib := s.loadStdlib()
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
	var location *lsp.Location
	name := s.GetTokenAtPosition(uri, position).Content
	if callName := findFunctionCallNameAtPosition(s.Ast[uri], position); callName != "" {
		name = callName
	}
	if resolved, ok := s.resolveDefinitionLocation(uri, name); ok {
		location = &resolved
	}

	return lsp.DefinitionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: location,
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
	stdlib := s.loadStdlib()
	resolver := func(name string) (FunctionSignature, bool) {
		return resolveSignatureForName(name, uri, s.Signatures[uri], stdlib)
	}
	inlayHints := GenerateInlayHints(s.Signatures[uri], s.Ast[uri], s.Tokens[uri], resolver)

	return lsp.InlayHintResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: inlayHints,
	}
}

package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

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
	mu             sync.RWMutex
	cacheMu        sync.Mutex
	Documents      map[string]string
	Ast            map[string][]p.Node
	Tokens         map[string][]l.Token
	Signatures     map[string][]FunctionSignature
	Resolved       map[string][]FunctionSignature
	IncludeOrigins map[string]map[string]string
	Diagnostics    map[string][]lsp.Diagnostic
	AstDirty       map[string]bool
	includeCache   map[string]includeCacheEntry
	stdlib         map[string]map[string][]FunctionSignature
	stdlibDecls    map[string]map[string][]StdlibDeclaration
	builtins       []FunctionSignature
	stdlibErr      error
	stdlibDeclsErr error
	builtinsErr    error
	stdlibLoaded   bool
	stdlibDeclsOk  bool
	builtinsLoaded bool

	stdlibDefinitionRoot  string
	stdlibDefinitionFiles map[string]stdlibDefinitionFile
	workspaceFolders      []string
}

type includeCacheEntry struct {
	ModTimeUnixNano int64
	Size            int64
	Ast             []p.Node
	Signatures      []FunctionSignature
}

type inlayCallResolution struct {
	Signature   FunctionSignature
	OriginLabel string
	ShowOrigin  bool
}

const maxOriginResolutionCallNames = 200

func NewState() State {
	stdlibDefinitionPruneOnce.Do(func() {
		_ = pruneStdlibDefinitionRoots(os.TempDir(), processPIDActive)
	})

	return State{
		Documents:             map[string]string{},
		Ast:                   map[string][]p.Node{},
		Tokens:                map[string][]l.Token{},
		Signatures:            map[string][]FunctionSignature{},
		Resolved:              map[string][]FunctionSignature{},
		IncludeOrigins:        map[string]map[string]string{},
		Diagnostics:           map[string][]lsp.Diagnostic{},
		AstDirty:              map[string]bool{},
		includeCache:          map[string]includeCacheEntry{},
		stdlibDefinitionFiles: map[string]stdlibDefinitionFile{},
	}
}

func (s *State) SetWorkspaceFolders(folders []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspaceFolders = folders
}

func (s *State) WorkspaceFolders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspaceFolders
}

func DetectWorkspaceRootFromDocument(uri string) string {
	docPath := uriToPath(uri)
	if docPath == "" {
		return ""
	}
	dir := filepath.Dir(docPath)
	for {
		scriptsPath := filepath.Join(dir, "scripts")
		if _, err := os.Stat(scriptsPath); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func (s *State) Close() error {
	if s.stdlibDefinitionRoot == "" {
		return nil
	}

	root := s.stdlibDefinitionRoot
	s.stdlibDefinitionRoot = ""
	s.stdlibDefinitionFiles = map[string]stdlibDefinitionFile{}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("remove stdlib definition root: %w", err)
	}
	return nil
}

func (s *State) DocumentText(uri string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Documents[uri]
}

func (s *State) OpenDocument(uri, text string) {
	s.mu.Lock()
	s.Documents[uri] = text
	s.updateAstLocked(uri)
	s.mu.Unlock()
}

func (s *State) UpdateDocument(uri, text string) {
	s.mu.Lock()
	if existing, ok := s.Documents[uri]; ok && existing == text {
		s.mu.Unlock()
		return
	}
	s.Documents[uri] = text
	s.updateAstLocked(uri)
	s.mu.Unlock()
}

func (s *State) ApplyIncrementalChange(uri string, change lsp.TextDocumentContentChangeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if change.Range == nil {
		s.Documents[uri] = change.Text
		s.updateAstLocked(uri)
		return
	}

	existing := s.Documents[uri]
	updated, ok := applyRangeChange(existing, change.Range, change.Text)
	if !ok {
		s.Documents[uri] = change.Text
		s.updateAstLocked(uri)
		return
	}

	s.Documents[uri] = updated
	s.AstDirty[uri] = true
	s.Diagnostics[uri] = nil
	delete(s.Resolved, uri)
	delete(s.IncludeOrigins, uri)
}

func (s *State) EnsureParsed(uri string) {
	s.mu.Lock()
	if s.AstDirty[uri] {
		s.updateAstLocked(uri)
		s.AstDirty[uri] = false
	}
	s.mu.Unlock()
}

func (s *State) ClearCaches(uri string) {
	delete(s.Resolved, uri)
	delete(s.IncludeOrigins, uri)
}

func applyRangeChange(doc string, range_ *lsp.Range, newText string) (string, bool) {
	if range_ == nil {
		return newText, true
	}

	startLine := range_.Start.Line
	startChar := range_.Start.Character
	endLine := range_.End.Line
	endChar := range_.End.Character

	lines := strings.Split(doc, "\n")

	if startLine >= len(lines) || endLine >= len(lines) {
		return "", false
	}

	lineLen := len(lines[startLine])
	if startChar > lineLen {
		return "", false
	}
	if endChar > len(lines[endLine]) {
		return "", false
	}

	start := lines[startLine][:startChar]
	end := lines[endLine][endChar:]

	if startLine == endLine {
		lines[startLine] = start + newText + end
	} else {
		lines[startLine] = start + newText
		lines = append(lines[:startLine+1], lines[endLine+1:]...)
		if endLine > startLine {
			lines[startLine] += end
		}
	}
	return strings.Join(lines, "\n"), true
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
	s.mu.RLock()
	workspaceFolders := s.workspaceFolders
	s.mu.RUnlock()
	resolvedPath, ok := resolveIncludePath(uri, filePath, workspaceFolders)
	if !ok {
		return fmt.Errorf("resolve include path: %s", filePath)
	}

	entry, err := s.getParsedInclude(resolvedPath)
	if err != nil {
		return fmt.Errorf("parse include file %q: %w", resolvedPath, err)
	}
	s.mu.Lock()
	s.Signatures[uri] = mergeSignatures(s.Signatures[uri], entry.Signatures)
	s.mu.Unlock()
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

func resolveIncludePath(uri, includePath string, workspaceFolders []string) (string, bool) {
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

	for _, workspaceRoot := range workspaceFolders {
		workspacePath := uriToPath(workspaceRoot)
		if workspacePath == "" {
			continue
		}
		candidate := filepath.Join(workspacePath, relativePath)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateAstLocked(uri)
}

func (s *State) updateAstLocked(uri string) {
	if err := s.parseAndStoreLocked(uri); err != nil {
		s.Diagnostics[uri] = []lsp.Diagnostic{parseFailureDiagnostic(err)}
		delete(s.Resolved, uri)
		delete(s.IncludeOrigins, uri)
		return
	}
	delete(s.Resolved, uri)
	delete(s.IncludeOrigins, uri)
}

func (s *State) parseAndStoreLocked(uri string) error {
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

func (s *State) mergeBuiltins(signatures []FunctionSignature) []FunctionSignature {
	builtins, err := s.loadBuiltins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR LOADING BUILTIN SIGNATURES: %v\n", err)
		return signatures
	}
	return mergeSignatures(signatures, builtins)
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

func (s *State) loadStdlibDeclarations() map[string]map[string][]StdlibDeclaration {
	if s.stdlibDeclsOk {
		return s.stdlibDecls
	}
	s.stdlibDecls, s.stdlibDeclsErr = StdlibDeclarations()
	s.stdlibDeclsOk = true
	if s.stdlibDeclsErr != nil {
		fmt.Fprintf(os.Stderr, "ERROR LOADING STDLIB DECLARATIONS: %v\n", s.stdlibDeclsErr)
	}
	return s.stdlibDecls
}

func (s *State) applyIncludes(signatures []FunctionSignature, uri string, includePaths []string, stdlibGroup string, stdlib map[string]map[string][]FunctionSignature) []FunctionSignature {
	s.mu.RLock()
	workspaceFolders := s.workspaceFolders
	s.mu.RUnlock()

	for _, includePath := range includePaths {
		key := normalizeIncludeKey(includePath)
		if key == "" {
			continue
		}

		if sigs, ok := lookupStdlibSignatures(stdlib, stdlibGroup, key); ok {
			signatures = mergeSignatures(signatures, sigs)
			continue
		}

		resolvedPath, ok := resolveIncludePath(uri, includePath, workspaceFolders)
		if !ok {
			continue
		}
		entry, err := s.getParsedInclude(resolvedPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR APPLYING INCLUDE %q: %v\n", includePath, err)
			continue
		}
		signatures = mergeSignatures(signatures, entry.Signatures)
	}

	return signatures
}

func (s *State) resolvedSignatures(uri string) []FunctionSignature {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if cached, ok := s.Resolved[uri]; ok {
		return cached
	}

	resolved := make([]FunctionSignature, 0, len(s.Signatures[uri]))
	resolved = append(resolved, s.Signatures[uri]...)
	resolved = s.mergeBuiltins(resolved)

	stdlib := s.loadStdlib()
	includePaths := collectIncludePaths(s.Ast[uri])
	stdlibGroup := guessStdlibGroup(uri)
	resolved = s.applyIncludes(resolved, uri, includePaths, stdlibGroup, stdlib)

	s.Resolved[uri] = resolved
	return resolved
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
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.EnsureParsed(uri)
	s.mu.RLock()
	defer s.mu.RUnlock()
	output := strings.Builder{}
	signatures := s.resolvedSignatures(uri)

	token := s.GetTokenAtPosition(uri, position)
	if token.Type == l.SYMBOL {
		stdlib := s.loadStdlib()
		name := token.Content
		sig, ok := resolveSignatureForName(name, uri, signatures, stdlib)
		if !ok {
			name = findFunctionCallNameAtPosition(s.Ast[uri], position)
			sig, ok = resolveSignatureForName(name, uri, signatures, stdlib)
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
	s.EnsureParsed(uri)
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.EnsureParsed(uri)
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.EnsureParsed(uri)
	s.mu.RLock()
	defer s.mu.RUnlock()
	stdlib := s.loadStdlib()
	signatures := s.resolvedSignatures(uri)
	signatureByName := buildSignatureMap(signatures)
	localDecls := buildLocalDeclarationSet(s.Ast[uri])
	builtinSet := s.buildBuiltinSet()
	originByName := map[string]string{}
	if countFunctionCallNames(s.Ast[uri]) <= maxOriginResolutionCallNames {
		originByName = s.buildIncludeOriginIndex(uri, stdlib)
	}
	resolutionCache := map[string]inlayCallResolution{}
	missingResolution := map[string]struct{}{}
	resolver := func(name string) (InlayHintResolution, bool) {
		resolved, ok := resolveInlayCallFast(uri, name, stdlib, signatureByName, localDecls, builtinSet, originByName, resolutionCache, missingResolution)
		if !ok {
			return InlayHintResolution{}, false
		}
		return InlayHintResolution{
			Signature:   resolved.Signature,
			OriginLabel: resolved.OriginLabel,
			ShowOrigin:  resolved.ShowOrigin,
		}, true
	}
	lines := strings.Split(s.Documents[uri], "\n")
	inlayHints := GenerateInlayHints(signatures, s.Ast[uri], s.Tokens[uri], resolver, lines)
	inlayHints = append(inlayHints, generateSelfContextInlayHints(s.Ast[uri], s.Tokens[uri])...)

	return lsp.InlayHintResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: inlayHints,
	}
}

func generateSelfContextInlayHints(nodes []p.Node, tokens []l.Token) []lsp.InlayHint {
	if len(nodes) == 0 || len(tokens) == 0 {
		return []lsp.InlayHint{}
	}

	receiversByFunction := collectFunctionReceivers(nodes)
	if len(receiversByFunction) == 0 {
		return []lsp.InlayHint{}
	}

	tokenIndex := indexTokensByLine(tokens)
	declarations := collectFunctionDeclarations(nodes)
	hints := make([]lsp.InlayHint, 0)

	for _, declaration := range declarations {
		name := strings.ToLower(strings.TrimSpace(declaration.Data.FunctionName))
		if name == "" {
			continue
		}

		receivers, ok := receiversByFunction[name]
		if !ok || len(receivers) == 0 {
			continue
		}

		resolvedReceivers := make([]string, 0, len(receivers))
		for _, receiver := range receivers {
			if receiver == "" || strings.EqualFold(receiver, "self") {
				continue
			}
			resolvedReceivers = append(resolvedReceivers, receiver)
		}
		if len(resolvedReceivers) == 0 {
			continue
		}

		labelReceivers := resolvedReceivers
		overflow := false
		if len(labelReceivers) > 3 {
			labelReceivers = labelReceivers[:3]
			overflow = true
		}

		startLine := declaration.Line
		if startLine <= 0 {
			continue
		}
		if !declarationUsesSelfContext(declaration) {
			continue
		}

		hintLine := startLine - 1
		hintCol := functionDeclarationHintAnchor(declaration, tokenIndex[startLine])
		hints = append(hints, lsp.InlayHint{
			Position: lsp.Position{Line: hintLine, Character: hintCol},
			Label:    formatSelfContextLabel(labelReceivers, overflow),
		})
	}

	return hints
}

func isSelfContextSymbol(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	return strings.EqualFold(trimmed, "self") || strings.HasPrefix(strings.ToLower(trimmed), "self.")
}

func formatSelfContextLabel(receivers []string, overflow bool) string {
	if len(receivers) == 0 {
		return " ->"
	}
	parts := make([]string, 0, len(receivers)+1)
	for _, receiver := range receivers {
		parts = append(parts, receiver)
	}
	if overflow {
		parts = append(parts, "...")
	}
	return " -> " + strings.Join(parts, ", ")
}

func selfContextHintAnchor(token l.Token) int {
	if token.EndCol > 0 {
		return token.EndCol
	}
	if token.Col > 0 {
		return token.Col - 1
	}
	return 0
}

func declarationUsesSelfContext(declaration p.Node) bool {
	if len(declaration.Children) == 0 {
		return false
	}

	var walk func([]p.Node) bool
	walk = func(nodes []p.Node) bool {
		for _, node := range nodes {
			if node.Type == "function_declaration" {
				continue
			}
			if node.Type == "function_call" && isSelfContextSymbol(node.Data.Method) {
				return true
			}
			if len(node.Children) > 0 && walk(node.Children) {
				return true
			}
		}
		return false
	}

	return walk(declaration.Children)
}

func functionDeclarationHintAnchor(declaration p.Node, lineTokens []l.Token) int {
	name := strings.TrimSpace(declaration.Data.FunctionName)
	if name == "" {
		if declaration.Col > 0 {
			return declaration.Col - 1
		}
		return 0
	}

	startCol := declaration.Col - 1
	if startCol < 0 {
		startCol = 0
	}

	closestBeforeCol := -1
	closestBeforeDistance := 0
	for i, tok := range lineTokens {
		if tok.Type != l.SYMBOL || !strings.EqualFold(strings.TrimSpace(tok.Content), name) {
			continue
		}

		candidateCol := functionDeclarationParamsEndAnchor(lineTokens, i)
		candidateStart := tok.Col - 1
		if candidateStart >= startCol {
			return candidateCol
		}

		distance := startCol - candidateStart
		if closestBeforeCol < 0 || distance < closestBeforeDistance {
			closestBeforeCol = candidateCol
			closestBeforeDistance = distance
		}
	}

	if closestBeforeCol >= 0 {
		return closestBeforeCol
	}

	if declaration.Col > 0 {
		return declaration.Col - 1 + len(name)
	}
	return len(name)
}

func functionDeclarationParamsEndAnchor(lineTokens []l.Token, symbolIndex int) int {
	depth := 0
	seenOpen := false

	for i := symbolIndex + 1; i < len(lineTokens); i++ {
		tok := lineTokens[i]
		switch tok.Type {
		case l.OPEN_PAREN:
			seenOpen = true
			depth++
		case l.CLOSE_PAREN:
			if seenOpen {
				depth--
				if depth == 0 {
					return selfContextHintAnchor(tok)
				}
			}
		case l.NEWLINE, l.TERMINATOR:
			if seenOpen {
				return selfContextHintAnchor(tok)
			}
		}
	}

	return selfContextHintAnchor(lineTokens[symbolIndex])
}

func collectFunctionReceivers(nodes []p.Node) map[string][]string {
	receiverSets := map[string]map[string]struct{}{}
	type selfReceiverEdge struct {
		caller string
		callee string
	}
	selfEdges := []selfReceiverEdge{}

	var walk func(items []p.Node, currentFunction string)
	walk = func(items []p.Node, currentFunction string) {
		for _, node := range items {
			if node.Type == "function_declaration" {
				nextFunction := strings.ToLower(strings.TrimSpace(node.Data.FunctionName))
				walk(node.Children, nextFunction)
				continue
			}

			if node.Type == "function_call" {
				name := strings.ToLower(strings.TrimSpace(node.Data.FunctionName))
				receiver := strings.TrimSpace(node.Data.Method)
				if name != "" && receiver != "" {
					if strings.EqualFold(receiver, "self") {
						if currentFunction != "" {
							selfEdges = append(selfEdges, selfReceiverEdge{caller: currentFunction, callee: name})
						}
					} else {
						if _, ok := receiverSets[name]; !ok {
							receiverSets[name] = map[string]struct{}{}
						}
						receiverSets[name][receiver] = struct{}{}
					}
				}
			}
			if len(node.Children) > 0 {
				walk(node.Children, currentFunction)
			}
		}
	}

	walk(nodes, "")

	changed := true
	for changed {
		changed = false
		for _, edge := range selfEdges {
			callerReceivers, ok := receiverSets[edge.caller]
			if !ok || len(callerReceivers) == 0 {
				continue
			}
			calleeReceivers, ok := receiverSets[edge.callee]
			if !ok {
				calleeReceivers = map[string]struct{}{}
				receiverSets[edge.callee] = calleeReceivers
			}
			for receiver := range callerReceivers {
				if _, exists := calleeReceivers[receiver]; exists {
					continue
				}
				calleeReceivers[receiver] = struct{}{}
				changed = true
			}
		}
	}

	receiversByFunction := map[string][]string{}
	for functionName, receivers := range receiverSets {
		if len(receivers) == 0 {
			continue
		}

		sortedReceivers := make([]string, 0, len(receivers))
		for receiver := range receivers {
			sortedReceivers = append(sortedReceivers, receiver)
		}
		sort.Strings(sortedReceivers)
		receiversByFunction[functionName] = sortedReceivers
	}

	return receiversByFunction
}

func collectFunctionDeclarations(nodes []p.Node) []p.Node {
	declarations := []p.Node{}
	var walk func([]p.Node)
	walk = func(items []p.Node) {
		for _, node := range items {
			if node.Type == "function_declaration" {
				declarations = append(declarations, node)
			}
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}

	walk(nodes)
	return declarations
}

func resolveInlayCallFast(
	uri, name string,
	stdlib map[string]map[string][]FunctionSignature,
	signatureByName map[string]FunctionSignature,
	localDecls map[string]struct{},
	builtinSet map[string]struct{},
	originByName map[string]string,
	resolutionCache map[string]inlayCallResolution,
	missingResolution map[string]struct{},
) (inlayCallResolution, bool) {
	if name == "" {
		return inlayCallResolution{}, false
	}

	cacheKey := strings.ToLower(strings.TrimSpace(name))
	if cached, ok := resolutionCache[cacheKey]; ok {
		return cached, true
	}
	if _, missing := missingResolution[cacheKey]; missing {
		return inlayCallResolution{}, false
	}

	sig, ok := resolveSignatureForNameFast(uri, name, stdlib, signatureByName)
	if !ok {
		missingResolution[cacheKey] = struct{}{}
		return inlayCallResolution{}, false
	}

	if _, _, qualified := splitQualifiedName(name); qualified {
		resolved := inlayCallResolution{Signature: sig}
		resolutionCache[cacheKey] = resolved
		return resolved, true
	}

	if _, ok := localDecls[strings.ToLower(name)]; ok {
		resolved := inlayCallResolution{Signature: sig}
		resolutionCache[cacheKey] = resolved
		return resolved, true
	}

	if _, ok := builtinSet[strings.ToLower(name)]; ok {
		resolved := inlayCallResolution{Signature: sig}
		resolutionCache[cacheKey] = resolved
		return resolved, true
	}

	if originLabel, ok := originByName[strings.ToLower(name)]; ok {
		resolved := inlayCallResolution{Signature: sig, OriginLabel: originLabel + "::", ShowOrigin: true}
		resolutionCache[cacheKey] = resolved
		return resolved, true
	}

	resolved := inlayCallResolution{Signature: sig}
	resolutionCache[cacheKey] = resolved
	return resolved, true
}

func resolveSignatureForNameFast(uri, name string, stdlib map[string]map[string][]FunctionSignature, signatureByName map[string]FunctionSignature) (FunctionSignature, bool) {
	if name == "" {
		return FunctionSignature{}, false
	}

	if sig, ok := signatureByName[strings.ToLower(name)]; ok {
		return sig, true
	}

	if _, funcName, ok := splitQualifiedName(name); ok {
		if sig, ok := signatureByName[strings.ToLower(funcName)]; ok {
			return sig, true
		}
		return resolveQualifiedSignature(stdlib, uri, name)
	}

	return FunctionSignature{}, false
}

func buildSignatureMap(signatures []FunctionSignature) map[string]FunctionSignature {
	indexed := make(map[string]FunctionSignature, len(signatures))
	for _, sig := range signatures {
		key := strings.ToLower(strings.TrimSpace(sig.Name))
		if key == "" {
			continue
		}
		if _, exists := indexed[key]; !exists {
			indexed[key] = sig
		}
	}
	return indexed
}

func buildLocalDeclarationSet(nodes []p.Node) map[string]struct{} {
	local := map[string]struct{}{}
	collectLocalDeclarationNames(nodes, local)
	return local
}

func collectLocalDeclarationNames(nodes []p.Node, out map[string]struct{}) {
	for _, node := range nodes {
		if node.Type == "function_declaration" {
			name := strings.ToLower(strings.TrimSpace(node.Data.FunctionName))
			if name != "" {
				out[name] = struct{}{}
			}
		}
		if len(node.Children) > 0 {
			collectLocalDeclarationNames(node.Children, out)
		}
	}
}

func (s *State) buildBuiltinSet() map[string]struct{} {
	builtinSet := map[string]struct{}{}
	builtins, err := s.loadBuiltins()
	if err != nil {
		return builtinSet
	}
	for _, sig := range builtins {
		name := strings.ToLower(strings.TrimSpace(sig.Name))
		if name == "" {
			continue
		}
		builtinSet[name] = struct{}{}
	}
	return builtinSet
}

func (s *State) buildIncludeOriginIndex(uri string, stdlib map[string]map[string][]FunctionSignature) map[string]string {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if cached, ok := s.IncludeOrigins[uri]; ok {
		return cached
	}

	index := map[string]string{}
	visitedLocal := map[string]bool{}
	visitedStdlib := map[string]bool{}
	s.addIncludeOriginsRecursive(uri, collectIncludePaths(s.Ast[uri]), stdlib, visitedLocal, visitedStdlib, index)
	s.IncludeOrigins[uri] = index
	return index
}

func (s *State) addIncludeOriginsRecursive(uri string, includePaths []string, stdlib map[string]map[string][]FunctionSignature, visitedLocal map[string]bool, visitedStdlib map[string]bool, index map[string]string) {
	stdlibGroup := guessStdlibGroup(uri)
	s.mu.RLock()
	workspaceFolders := s.workspaceFolders
	s.mu.RUnlock()

	for _, includePath := range includePaths {
		key := normalizeIncludeKey(includePath)
		label := ""
		if key != "" {
			label = slashPathToGsc(key)
		}

		if key != "" {
			if _, seen := visitedStdlib[key]; !seen {
				visitedStdlib[key] = true
				if sigs, ok := lookupStdlibSignatures(stdlib, stdlibGroup, key); ok {
					addOriginSignatures(index, sigs, label)
				}
			}
		}

		resolvedPath, ok := resolveIncludePath(uri, includePath, workspaceFolders)
		if !ok {
			continue
		}
		resolvedPath = filepath.Clean(resolvedPath)
		if visitedLocal[resolvedPath] {
			continue
		}
		visitedLocal[resolvedPath] = true

		entry, err := s.getParsedInclude(resolvedPath)
		if err != nil {
			continue
		}

		addOriginSignatures(index, entry.Signatures, label)

		includeURI := pathToURI(resolvedPath)
		nestedIncludes := collectIncludePaths(entry.Ast)
		s.addIncludeOriginsRecursive(includeURI, nestedIncludes, stdlib, visitedLocal, visitedStdlib, index)
	}
}

func addOriginSignatures(index map[string]string, signatures []FunctionSignature, label string) {
	if label == "" {
		return
	}
	for _, sig := range signatures {
		name := strings.ToLower(strings.TrimSpace(sig.Name))
		if name == "" {
			continue
		}
		if _, exists := index[name]; exists {
			continue
		}
		index[name] = label
	}
}

func countFunctionCallNames(nodes []p.Node) int {
	seen := map[string]struct{}{}
	collectFunctionCallNames(nodes, seen)
	return len(seen)
}

func collectFunctionCallNames(nodes []p.Node, out map[string]struct{}) {
	for _, node := range nodes {
		if node.Type == "function_call" {
			name := strings.ToLower(strings.TrimSpace(functionCallName(node)))
			if name != "" {
				out[name] = struct{}{}
			}
		}
		if len(node.Children) > 0 {
			collectFunctionCallNames(node.Children, out)
		}
	}
}

func (s *State) resolveInlayCall(uri, name string, signatures []FunctionSignature, stdlib map[string]map[string][]FunctionSignature) (inlayCallResolution, bool) {
	if name == "" {
		return inlayCallResolution{}, false
	}

	if sig, ok := resolveSignatureForName(name, uri, signatures, stdlib); !ok {
		return inlayCallResolution{}, false
	} else if _, _, qualified := splitQualifiedName(name); qualified {
		return inlayCallResolution{Signature: sig}, true
	}

	if findFunctionDeclarationNodeByName(s.Ast[uri], name) != nil {
		sig, _ := resolveSignatureForName(name, uri, signatures, stdlib)
		return inlayCallResolution{Signature: sig}, true
	}

	builtins, err := s.loadBuiltins()
	if err == nil {
		if _, ok := findSignatureByName(builtins, name); ok {
			sig, _ := resolveSignatureForName(name, uri, signatures, stdlib)
			return inlayCallResolution{Signature: sig}, true
		}
	}

	originLabel, ok := s.resolveIncludeOriginLabel(uri, name, stdlib)
	if !ok {
		sig, ok := resolveSignatureForName(name, uri, signatures, stdlib)
		if !ok {
			return inlayCallResolution{}, false
		}
		return inlayCallResolution{Signature: sig}, true
	}

	sig, ok := resolveSignatureForName(name, uri, signatures, stdlib)
	if !ok {
		return inlayCallResolution{}, false
	}

	return inlayCallResolution{
		Signature:   sig,
		OriginLabel: originLabel + "::",
		ShowOrigin:  true,
	}, true
}

func (s *State) resolveIncludeOriginLabel(uri, functionName string, stdlib map[string]map[string][]FunctionSignature) (string, bool) {
	visitedLocal := map[string]bool{}
	visitedStdlib := map[string]bool{}
	includePaths := collectIncludePaths(s.Ast[uri])
	return s.resolveIncludeOriginLabelRecursive(uri, includePaths, functionName, stdlib, visitedLocal, visitedStdlib)
}

func (s *State) resolveIncludeOriginLabelRecursive(uri string, includePaths []string, functionName string, stdlib map[string]map[string][]FunctionSignature, visitedLocal map[string]bool, visitedStdlib map[string]bool) (string, bool) {
	stdlibGroup := guessStdlibGroup(uri)
	s.mu.RLock()
	workspaceFolders := s.workspaceFolders
	s.mu.RUnlock()

	for _, includePath := range includePaths {
		key := normalizeIncludeKey(includePath)
		if key != "" {
			if _, seen := visitedStdlib[key]; !seen {
				visitedStdlib[key] = true
				if sigs, ok := lookupStdlibSignatures(stdlib, stdlibGroup, key); ok {
					if _, found := findSignatureByName(sigs, functionName); found {
						return slashPathToGsc(key), true
					}
				}
			}
		}

		resolvedPath, ok := resolveIncludePath(uri, includePath, workspaceFolders)
		if !ok {
			continue
		}
		resolvedPath = filepath.Clean(resolvedPath)
		if visitedLocal[resolvedPath] {
			continue
		}
		visitedLocal[resolvedPath] = true

		entry, err := s.getParsedInclude(resolvedPath)
		if err != nil {
			continue
		}

		if findFunctionDeclarationNodeByName(entry.Ast, functionName) != nil {
			if key != "" {
				return slashPathToGsc(key), true
			}
		}

		includeURI := pathToURI(resolvedPath)
		nestedIncludes := collectIncludePaths(entry.Ast)
		if label, ok := s.resolveIncludeOriginLabelRecursive(includeURI, nestedIncludes, functionName, stdlib, visitedLocal, visitedStdlib); ok {
			return label, true
		}
	}

	return "", false
}

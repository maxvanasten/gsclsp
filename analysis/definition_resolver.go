package analysis

import (
	"path/filepath"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	p "github.com/maxvanasten/gscp/parser"
)

func (s *State) resolveDefinitionLocation(uri, name string) (lsp.Location, bool) {
	if name == "" {
		return lsp.Location{}, false
	}

	workspaceFolders := s.WorkspaceFolders()

	if qualifier, funcName, ok := splitQualifiedName(name); ok {
		resolvedPath, ok := resolveIncludePath(uri, qualifier, workspaceFolders)
		if !ok {
			return s.resolveStdlibDefinitionLocation(uri, name)
		}
		return s.findDefinitionInFile(pathToURI(resolvedPath), funcName)
	}

	if loc, ok := findDefinitionInNodes(uri, s.Ast[uri], name); ok {
		return loc, true
	}

	visited := map[string]bool{}
	includePaths := collectIncludePaths(s.Ast[uri])
	if loc, ok := s.resolveDefinitionFromIncludes(uri, includePaths, name, visited); ok {
		return loc, true
	}

	return s.resolveStdlibDefinitionLocation(uri, name)
}

func (s *State) resolveStdlibDefinitionLocation(uri, name string) (lsp.Location, bool) {
	stdlib := s.loadStdlib()
	if stdlib == nil {
		return lsp.Location{}, false
	}
	declarations := s.loadStdlibDeclarations()
	stdlibGroup := guessStdlibGroup(uri)

	if qualifier, funcName, ok := splitQualifiedName(name); ok {
		key := normalizeIncludeKey(qualifier)
		if key == "" {
			return lsp.Location{}, false
		}
		sigs, group, ok := lookupStdlibSignaturesWithGroup(stdlib, stdlibGroup, key)
		if !ok {
			return lsp.Location{}, false
		}
		if _, ok := findSignatureByName(sigs, funcName); !ok {
			return lsp.Location{}, false
		}
		decls, _ := lookupStdlibDeclarationsForGroup(declarations, group, key)
		return s.stdlibDefinitionLocation(group, key, funcName, sigs, decls)
	}

	for _, includePath := range collectIncludePaths(s.Ast[uri]) {
		key := normalizeIncludeKey(includePath)
		if key == "" {
			continue
		}
		sigs, group, ok := lookupStdlibSignaturesWithGroup(stdlib, stdlibGroup, key)
		if !ok {
			continue
		}
		if _, ok := findSignatureByName(sigs, name); !ok {
			continue
		}
		decls, _ := lookupStdlibDeclarationsForGroup(declarations, group, key)
		return s.stdlibDefinitionLocation(group, key, name, sigs, decls)
	}

	return lsp.Location{}, false
}

func (s *State) stdlibDefinitionLocation(group, key, functionName string, signatures []FunctionSignature, declarations []StdlibDeclaration) (lsp.Location, bool) {
	file, err := s.ensureStdlibDefinitionFile(group, key, declarations, signatures)
	if err != nil {
		return lsp.Location{}, false
	}

	rangeValue, ok := file.Ranges[strings.ToLower(functionName)]
	if !ok {
		return lsp.Location{}, false
	}

	return lsp.Location{
		URI:   file.URI,
		Range: rangeValue,
	}, true
}

func lookupStdlibSignaturesWithGroup(stdlib map[string]map[string][]FunctionSignature, stdlibGroup, key string) ([]FunctionSignature, string, bool) {
	if stdlib == nil {
		return nil, "", false
	}
	if stdlibGroup != "" {
		if sigs, ok := stdlib[stdlibGroup][key]; ok {
			return sigs, stdlibGroup, true
		}
	}
	if sigs, ok := stdlib["mp"][key]; ok {
		return sigs, "mp", true
	}
	if sigs, ok := stdlib["zm"][key]; ok {
		return sigs, "zm", true
	}
	return nil, "", false
}

func lookupStdlibDeclarationsForGroup(declarations map[string]map[string][]StdlibDeclaration, group, key string) ([]StdlibDeclaration, bool) {
	if declarations == nil {
		return nil, false
	}
	if group != "" {
		if decls, ok := declarations[group][key]; ok {
			return decls, true
		}
	}
	return nil, false
}

func (s *State) resolveDefinitionFromIncludes(uri string, includePaths []string, functionName string, visited map[string]bool) (lsp.Location, bool) {
	workspaceFolders := s.WorkspaceFolders()

	for _, includePath := range includePaths {
		resolvedPath, ok := resolveIncludePath(uri, includePath, workspaceFolders)
		if !ok {
			continue
		}
		resolvedPath = filepath.Clean(resolvedPath)
		if visited[resolvedPath] {
			continue
		}
		visited[resolvedPath] = true

		includeURI := pathToURI(resolvedPath)
		entry, err := s.getParsedInclude(resolvedPath)
		if err != nil {
			continue
		}

		if loc, ok := findDefinitionInNodes(includeURI, entry.Ast, functionName); ok {
			return loc, true
		}

		nestedIncludes := collectIncludePaths(entry.Ast)
		if loc, ok := s.resolveDefinitionFromIncludes(includeURI, nestedIncludes, functionName, visited); ok {
			return loc, true
		}
	}

	return lsp.Location{}, false
}

func (s *State) findDefinitionInFile(uri, functionName string) (lsp.Location, bool) {
	path := uriToPath(uri)
	if path == "" {
		return lsp.Location{}, false
	}

	entry, err := s.getParsedInclude(path)
	if err != nil {
		return lsp.Location{}, false
	}

	return findDefinitionInNodes(uri, entry.Ast, functionName)
}

func findDefinitionInNodes(uri string, nodes []p.Node, functionName string) (lsp.Location, bool) {
	decl := findFunctionDeclarationNodeByName(nodes, functionName)
	if decl == nil {
		return lsp.Location{}, false
	}

	startLine := decl.Line - 1
	if startLine < 0 {
		startLine = 0
	}
	startCol := decl.Col - 1
	if startCol < 0 {
		startCol = 0
	}
	endCol := startCol + decl.Length
	if endCol < startCol {
		endCol = startCol
	}

	return lsp.Location{
		URI: uri,
		Range: lsp.Range{
			Start: lsp.Position{Line: startLine, Character: startCol},
			End:   lsp.Position{Line: startLine, Character: endCol},
		},
	}, true
}

func findFunctionDeclarationNodeByName(nodes []p.Node, name string) *p.Node {
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return nil
	}
	for i := range nodes {
		n := &nodes[i]
		if n.Type == "function_declaration" && strings.ToLower(n.Data.FunctionName) == needle {
			return n
		}
		if len(n.Children) > 0 {
			if found := findFunctionDeclarationNodeByName(n.Children, name); found != nil {
				return found
			}
		}
	}
	return nil
}

func pathToURI(path string) string {
	if path == "" {
		return ""
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, "/") {
		return "file://" + path
	}
	return "file:///" + path
}

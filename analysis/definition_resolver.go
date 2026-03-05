package analysis

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	p "github.com/maxvanasten/gscp/parser"
)

func (s *State) resolveDefinitionLocation(uri, name string) (lsp.Location, bool) {
	if name == "" {
		return lsp.Location{}, false
	}

	if qualifier, funcName, ok := splitQualifiedName(name); ok {
		resolvedPath, ok := resolveIncludePath(uri, qualifier)
		if !ok {
			return lsp.Location{}, false
		}
		return findDefinitionInFile(pathToURI(resolvedPath), funcName)
	}

	if loc, ok := findDefinitionInNodes(uri, s.Ast[uri], name); ok {
		return loc, true
	}

	visited := map[string]bool{}
	includePaths := collectIncludePaths(s.Ast[uri])
	return resolveDefinitionFromIncludes(uri, includePaths, name, visited)
}

func resolveDefinitionFromIncludes(uri string, includePaths []string, functionName string, visited map[string]bool) (lsp.Location, bool) {
	for _, includePath := range includePaths {
		resolvedPath, ok := resolveIncludePath(uri, includePath)
		if !ok {
			continue
		}
		resolvedPath = filepath.Clean(resolvedPath)
		if visited[resolvedPath] {
			continue
		}
		visited[resolvedPath] = true

		includeURI := pathToURI(resolvedPath)
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			continue
		}
		parseResult, err := Parse(string(data))
		if err != nil {
			continue
		}

		if loc, ok := findDefinitionInNodes(includeURI, parseResult.Ast, functionName); ok {
			return loc, true
		}

		nestedIncludes := collectIncludePaths(parseResult.Ast)
		if loc, ok := resolveDefinitionFromIncludes(includeURI, nestedIncludes, functionName, visited); ok {
			return loc, true
		}
	}

	return lsp.Location{}, false
}

func findDefinitionInFile(uri, functionName string) (lsp.Location, bool) {
	path := uriToPath(uri)
	if path == "" {
		return lsp.Location{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return lsp.Location{}, false
	}
	parseResult, err := Parse(string(data))
	if err != nil {
		return lsp.Location{}, false
	}

	return findDefinitionInNodes(uri, parseResult.Ast, functionName)
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

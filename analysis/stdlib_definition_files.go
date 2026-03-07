package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
)

type stdlibDefinitionFile struct {
	URI    string
	Path   string
	Ranges map[string]lsp.Range
}

func (s *State) ensureStdlibDefinitionFile(group, key string, declarations []StdlibDeclaration, signatures []FunctionSignature) (stdlibDefinitionFile, error) {
	cacheKey := group + "::" + key
	if cached, ok := s.stdlibDefinitionFiles[cacheKey]; ok {
		return cached, nil
	}

	root, err := s.ensureStdlibDefinitionRoot()
	if err != nil {
		return stdlibDefinitionFile{}, err
	}

	path := filepath.Join(root, group, filepath.FromSlash(key)+".gsc")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return stdlibDefinitionFile{}, fmt.Errorf("mkdir stdlib definition path: %w", err)
	}

	entries := mergeDeclarationEntries(declarations, signatures)
	if len(entries) == 0 {
		return stdlibDefinitionFile{}, fmt.Errorf("no stdlib declarations for %s/%s", group, key)
	}

	content, ranges := renderStdlibDefinitionFile(entries)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return stdlibDefinitionFile{}, fmt.Errorf("write stdlib definition file: %w", err)
	}

	file := stdlibDefinitionFile{
		URI:    pathToURI(path),
		Path:   path,
		Ranges: ranges,
	}
	s.stdlibDefinitionFiles[cacheKey] = file
	return file, nil
}

func (s *State) ensureStdlibDefinitionRoot() (string, error) {
	if s.stdlibDefinitionRoot != "" {
		return s.stdlibDefinitionRoot, nil
	}

	root, err := os.MkdirTemp("", "gsclsp-stdlib-defs-")
	if err != nil {
		return "", fmt.Errorf("create stdlib definition root: %w", err)
	}
	s.stdlibDefinitionRoot = root
	return root, nil
}

func mergeDeclarationEntries(declarations []StdlibDeclaration, signatures []FunctionSignature) []StdlibDeclaration {
	merged := make([]StdlibDeclaration, 0, len(declarations)+len(signatures))
	seen := map[string]struct{}{}

	for _, decl := range declarations {
		name := strings.TrimSpace(decl.Name)
		if name == "" {
			continue
		}
		decl.Name = name
		key := signatureKey(FunctionSignature{Name: name, Arguments: decl.Arguments})
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, decl)
	}

	for _, sig := range signatures {
		key := signatureKey(sig)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, StdlibDeclaration{
			Name:        sig.Name,
			Arguments:   sig.Arguments,
			Declaration: renderFallbackDeclaration(sig),
		})
	}

	sort.SliceStable(merged, func(i, j int) bool {
		iName := strings.ToLower(merged[i].Name)
		jName := strings.ToLower(merged[j].Name)
		if iName == jName {
			return signatureKey(FunctionSignature{Name: merged[i].Name, Arguments: merged[i].Arguments}) < signatureKey(FunctionSignature{Name: merged[j].Name, Arguments: merged[j].Arguments})
		}
		return iName < jName
	})

	return merged
}

func renderStdlibDefinitionFile(entries []StdlibDeclaration) (string, map[string]lsp.Range) {
	ranges := map[string]lsp.Range{}
	content := strings.Builder{}
	line := 0

	for i, entry := range entries {
		decl := normalizeDeclarationText(entry)
		if decl == "" {
			decl = renderFallbackDeclaration(FunctionSignature{Name: entry.Name, Arguments: entry.Arguments})
		}

		offset := declarationNameOffset(decl, entry.Name)
		if offset >= 0 {
			relLine, relCol := offsetToLineCol(decl, offset)
			ranges[strings.ToLower(entry.Name)] = lsp.Range{
				Start: lsp.Position{Line: line + relLine, Character: relCol},
				End:   lsp.Position{Line: line + relLine, Character: relCol + len(entry.Name)},
			}
		}

		content.WriteString(decl)
		if i+1 < len(entries) {
			content.WriteString("\n\n")
			line += strings.Count(decl, "\n") + 2
		} else {
			content.WriteString("\n")
			line += strings.Count(decl, "\n") + 1
		}
	}

	return content.String(), ranges
}

func normalizeDeclarationText(entry StdlibDeclaration) string {
	text := strings.ReplaceAll(entry.Declaration, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return text
}

func declarationNameOffset(declaration, name string) int {
	lowerDeclaration := strings.ToLower(declaration)
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if lowerName == "" {
		return -1
	}
	if idx := strings.Index(lowerDeclaration, lowerName+"("); idx >= 0 {
		return idx
	}
	return strings.Index(lowerDeclaration, lowerName)
}

func offsetToLineCol(text string, offset int) (int, int) {
	if offset <= 0 {
		return 0, 0
	}
	line := 0
	col := 0
	for i := 0; i < len(text) && i < offset; i++ {
		if text[i] == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}

func renderFallbackDeclaration(sig FunctionSignature) string {
	args := strings.Join(sig.Arguments, ", ")
	return sig.Name + "(" + args + ") {\n}\n"
}

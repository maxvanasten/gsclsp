package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maxvanasten/gsclsp/analysis"
	p "github.com/maxvanasten/gscp/parser"
)

type outputBundle struct {
	MP map[string][]analysis.FunctionSignature `json:"mp"`
	ZM map[string][]analysis.FunctionSignature `json:"zm"`
}

type declarationOutputBundle struct {
	MP map[string][]analysis.StdlibDeclaration `json:"mp"`
	ZM map[string][]analysis.StdlibDeclaration `json:"zm"`
}

func main() {
	mpRoot := flag.String("mp-root", "", "Path to mp core root")
	zmRoot := flag.String("zm-root", "", "Path to zm core root")
	mpMapsRoot := flag.String("mp-maps-root", "", "Optional path to mp map scripts root")
	zmMapsRoot := flag.String("zm-maps-root", "", "Optional path to zm map scripts root")
	outPath := flag.String("out", "analysis/stdlib_signatures.json", "Output signatures json path")
	outDeclarationsPath := flag.String("out-declarations", "analysis/stdlib_declarations.json", "Output declarations json path")
	flag.Parse()

	if *mpRoot == "" || *zmRoot == "" {
		fmt.Fprintln(os.Stderr, "mp-root and zm-root are required")
		os.Exit(1)
	}

	mpSignatures, mpDeclarations, mpDuplicates, err := buildGroupSignatureMap([]signatureRoot{
		{Path: *mpRoot, Label: "mp core"},
		{Path: *mpMapsRoot, Label: "mp maps", MapRoot: true},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed building mp signatures: %v\n", err)
		os.Exit(1)
	}

	zmSignatures, zmDeclarations, zmDuplicates, err := buildGroupSignatureMap([]signatureRoot{
		{Path: *zmRoot, Label: "zm core"},
		{Path: *zmMapsRoot, Label: "zm maps", MapRoot: true},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed building zm signatures: %v\n", err)
		os.Exit(1)
	}

	reportDuplicateKeys("mp", mpDuplicates)
	reportDuplicateKeys("zm", zmDuplicates)

	signaturePayload, err := json.Marshal(outputBundle{MP: mpSignatures, ZM: zmSignatures})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal signatures json: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, signaturePayload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write signatures output: %v\n", err)
		os.Exit(1)
	}

	declarationsPayload, err := json.Marshal(declarationOutputBundle{MP: mpDeclarations, ZM: zmDeclarations})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal declarations json: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outDeclarationsPath, declarationsPayload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write declarations output: %v\n", err)
		os.Exit(1)
	}
}

type signatureRoot struct {
	Path      string
	Label     string
	KeyPrefix string
	MapRoot   bool
}

type duplicateSignatureKey struct {
	Key     string
	Sources []string
}

func buildGroupSignatureMap(roots []signatureRoot) (map[string][]analysis.FunctionSignature, map[string][]analysis.StdlibDeclaration, []duplicateSignatureKey, error) {
	signatures := map[string][]analysis.FunctionSignature{}
	declarations := map[string][]analysis.StdlibDeclaration{}
	keySources := map[string]map[string]struct{}{}
	duplicatesSeen := map[string]struct{}{}
	duplicates := []duplicateSignatureKey{}

	for _, root := range roots {
		if strings.TrimSpace(root.Path) == "" {
			continue
		}

		builtSignatures := map[string][]analysis.FunctionSignature{}
		builtDeclarations := map[string][]analysis.StdlibDeclaration{}
		builtSources := map[string][]string{}
		var err error

		if root.MapRoot {
			builtSignatures, builtDeclarations, builtSources, err = buildMapRootSignatureMap(root.Path)
		} else {
			builtSignatures, builtDeclarations, err = buildSignatureMap(root.Path, root.KeyPrefix)
			if err == nil {
				for key := range builtSignatures {
					builtSources[key] = []string{root.Label}
				}
			}
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", root.Label, err)
		}

		for key, add := range builtSignatures {
			signatures[key] = mergeSignatures(signatures[key], add)
			declarations[key] = mergeDeclarations(declarations[key], builtDeclarations[key])

			if _, ok := keySources[key]; !ok {
				keySources[key] = map[string]struct{}{}
			}
			for _, source := range builtSources[key] {
				keySources[key][source] = struct{}{}
			}

			if len(keySources[key]) > 1 {
				if _, seen := duplicatesSeen[key]; seen {
					continue
				}
				duplicatesSeen[key] = struct{}{}
				sources := mapKeysSorted(keySources[key])
				duplicates = append(duplicates, duplicateSignatureKey{Key: key, Sources: sources})
			}
		}
	}

	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Key < duplicates[j].Key
	})

	return signatures, declarations, duplicates, nil
}

func buildSignatureMap(root, keyPrefix string) (map[string][]analysis.FunctionSignature, map[string][]analysis.StdlibDeclaration, error) {
	signatures := map[string][]analysis.FunctionSignature{}
	declarations := map[string][]analysis.StdlibDeclaration{}
	root = filepath.Clean(root)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".gsc") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		key := normalizeKey(rel, keyPrefix)
		if key == "" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		source := string(data)
		parseResult, err := analysis.Parse(source)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		sigs := analysis.GenerateFunctionSignatures(parseResult.Ast)
		if len(sigs) == 0 {
			return nil
		}

		signatures[key] = mergeSignatures(signatures[key], sigs)
		declarations[key] = mergeDeclarations(declarations[key], extractFunctionDeclarations(parseResult.Ast, source))
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return signatures, declarations, nil
}

func buildMapRootSignatureMap(root string) (map[string][]analysis.FunctionSignature, map[string][]analysis.StdlibDeclaration, map[string][]string, error) {
	signatures := map[string][]analysis.FunctionSignature{}
	declarations := map[string][]analysis.StdlibDeclaration{}
	sources := map[string]map[string]struct{}{}

	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}

		mapDir := filepath.Join(root, d.Name())
		runtimeRoot, ok := findMapRuntimeRoot(mapDir)
		if !ok {
			continue
		}

		builtSignatures, builtDeclarations, err := buildSignatureMap(runtimeRoot, "maps/mp")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", d.Name(), err)
		}

		for key, add := range builtSignatures {
			signatures[key] = mergeSignatures(signatures[key], add)
			declarations[key] = mergeDeclarations(declarations[key], builtDeclarations[key])
			if _, exists := sources[key]; !exists {
				sources[key] = map[string]struct{}{}
			}
			sources[key][d.Name()] = struct{}{}
		}
	}

	flattenedSources := map[string][]string{}
	for key, sourceSet := range sources {
		flattenedSources[key] = mapKeysSorted(sourceSet)
	}

	return signatures, declarations, flattenedSources, nil
}

func extractFunctionDeclarations(nodes []p.Node, source string) []analysis.StdlibDeclaration {
	declarations := []analysis.StdlibDeclaration{}
	for _, node := range nodes {
		if node.Type == "function_declaration" {
			declaration := sliceNodeText(source, node)
			if declaration != "" {
				declarations = append(declarations, analysis.StdlibDeclaration{
					Name:        node.Data.FunctionName,
					Arguments:   functionArguments(node),
					Declaration: strings.TrimRight(declaration, "\n"),
				})
			}
		}
		if len(node.Children) > 0 {
			declarations = append(declarations, extractFunctionDeclarations(node.Children, source)...)
		}
	}
	return declarations
}

func sliceNodeText(source string, node p.Node) string {
	if node.Line <= 0 || node.Col <= 0 || node.Length <= 0 {
		return ""
	}

	start, ok := offsetFromLineCol(source, node.Line, node.Col)
	if !ok || start < 0 || start >= len(source) {
		return ""
	}

	end := start + node.Length
	if end > len(source) {
		end = len(source)
	}
	if end <= start {
		return ""
	}

	return source[start:end]
}

func offsetFromLineCol(source string, line, col int) (int, bool) {
	if line <= 0 || col <= 0 {
		return 0, false
	}
	lineIdx := 1
	lineStart := 0
	for i := 0; i < len(source) && lineIdx < line; i++ {
		if source[i] == '\n' {
			lineIdx++
			lineStart = i + 1
		}
	}
	if lineIdx != line {
		return 0, false
	}
	offset := lineStart + col - 1
	if offset < lineStart || offset > len(source) {
		return 0, false
	}
	return offset, true
}

func functionArguments(node p.Node) []string {
	args := []string{}
	if len(node.Children) > 0 {
		for _, c := range node.Children[0].Children {
			args = append(args, c.Data.VarName)
		}
	}
	return args
}

func findMapRuntimeRoot(mapDir string) (string, bool) {
	mapsDir := filepath.Join(mapDir, "maps")
	entries, err := os.ReadDir(mapsDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), "mp") {
			return filepath.Join(mapsDir, entry.Name()), true
		}
	}

	return "", false
}

func normalizeKey(path, keyPrefix string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.ToLower(path)
	path = strings.TrimSuffix(path, ".gsc")
	if keyPrefix != "" {
		prefix := strings.Trim(filepath.ToSlash(keyPrefix), "/")
		if prefix != "" {
			path = prefix + "/" + path
		}
	}
	return strings.TrimSpace(path)
}

func mergeSignatureMaps(base map[string][]analysis.FunctionSignature, add map[string][]analysis.FunctionSignature) {
	if len(add) == 0 {
		return
	}

	keys := make([]string, 0, len(add))
	for key := range add {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		base[key] = mergeSignatures(base[key], add[key])
	}
}

func mapKeysSorted(value map[string]struct{}) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func reportDuplicateKeys(group string, duplicates []duplicateSignatureKey) {
	if len(duplicates) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "detected %d duplicate stdlib keys in %s group\n", len(duplicates), group)
	for _, duplicate := range duplicates {
		fmt.Fprintf(os.Stderr, "duplicate key: %s (sources: %s)\n", duplicate.Key, strings.Join(duplicate.Sources, ", "))
	}
}

func mergeSignatures(base, add []analysis.FunctionSignature) []analysis.FunctionSignature {
	if len(add) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(add))
	for _, sig := range base {
		seen[signatureKey(sig)] = struct{}{}
	}
	for _, sig := range add {
		key := signatureKey(sig)
		if _, exists := seen[key]; exists {
			continue
		}
		base = append(base, sig)
		seen[key] = struct{}{}
	}

	return base
}

func mergeDeclarations(base, add []analysis.StdlibDeclaration) []analysis.StdlibDeclaration {
	if len(add) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(add))
	for _, decl := range base {
		key := signatureKey(analysis.FunctionSignature{Name: decl.Name, Arguments: decl.Arguments})
		seen[key] = struct{}{}
	}
	for _, decl := range add {
		key := signatureKey(analysis.FunctionSignature{Name: decl.Name, Arguments: decl.Arguments})
		if _, exists := seen[key]; exists {
			continue
		}
		base = append(base, decl)
		seen[key] = struct{}{}
	}

	return base
}

func signatureKey(sig analysis.FunctionSignature) string {
	parts := []string{sig.Name}
	parts = append(parts, sig.Arguments...)
	return strings.Join(parts, "\x1f")
}

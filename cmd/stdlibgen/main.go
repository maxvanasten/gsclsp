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
)

type outputBundle struct {
	MP map[string][]analysis.FunctionSignature `json:"mp"`
	ZM map[string][]analysis.FunctionSignature `json:"zm"`
}

func main() {
	mpRoot := flag.String("mp-root", "", "Path to mp core root")
	zmRoot := flag.String("zm-root", "", "Path to zm core root")
	mpMapsRoot := flag.String("mp-maps-root", "", "Optional path to mp map scripts root")
	zmMapsRoot := flag.String("zm-maps-root", "", "Optional path to zm map scripts root")
	outPath := flag.String("out", "analysis/stdlib_signatures.json", "Output json path")
	flag.Parse()

	if *mpRoot == "" || *zmRoot == "" {
		fmt.Fprintln(os.Stderr, "mp-root and zm-root are required")
		os.Exit(1)
	}

	mpMap, mpDuplicates, err := buildGroupSignatureMap([]signatureRoot{
		{Path: *mpRoot, Label: "mp core"},
		{Path: *mpMapsRoot, Label: "mp maps", MapRoot: true},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed building mp signatures: %v\n", err)
		os.Exit(1)
	}

	zmMap, zmDuplicates, err := buildGroupSignatureMap([]signatureRoot{
		{Path: *zmRoot, Label: "zm core"},
		{Path: *zmMapsRoot, Label: "zm maps", MapRoot: true},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed building zm signatures: %v\n", err)
		os.Exit(1)
	}

	reportDuplicateKeys("mp", mpDuplicates)
	reportDuplicateKeys("zm", zmDuplicates)

	bundle := outputBundle{MP: mpMap, ZM: zmMap}
	payload, err := json.Marshal(bundle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal json: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
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

func buildGroupSignatureMap(roots []signatureRoot) (map[string][]analysis.FunctionSignature, []duplicateSignatureKey, error) {
	output := map[string][]analysis.FunctionSignature{}
	keySources := map[string]map[string]struct{}{}
	duplicatesSeen := map[string]struct{}{}
	duplicates := []duplicateSignatureKey{}

	for _, root := range roots {
		if strings.TrimSpace(root.Path) == "" {
			continue
		}

		built := map[string][]analysis.FunctionSignature{}
		builtSources := map[string][]string{}
		var err error
		if root.MapRoot {
			built, builtSources, err = buildMapRootSignatureMap(root.Path)
		} else {
			built, err = buildSignatureMap(root.Path, root.KeyPrefix)
			if err == nil {
				for key := range built {
					builtSources[key] = []string{root.Label}
				}
			}
		}
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", root.Label, err)
		}

		for key, add := range built {
			output[key] = mergeSignatures(output[key], add)

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

	return output, duplicates, nil
}

func buildSignatureMap(root, keyPrefix string) (map[string][]analysis.FunctionSignature, error) {
	output := map[string][]analysis.FunctionSignature{}
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

		parseResult, err := analysis.Parse(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		signatures := analysis.GenerateFunctionSignatures(parseResult.Ast)
		if len(signatures) == 0 {
			return nil
		}

		output[key] = mergeSignatures(output[key], signatures)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return output, nil
}

func buildMapRootSignatureMap(root string) (map[string][]analysis.FunctionSignature, map[string][]string, error) {
	output := map[string][]analysis.FunctionSignature{}
	sources := map[string]map[string]struct{}{}

	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
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

		built, err := buildSignatureMap(runtimeRoot, "maps/mp")
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", d.Name(), err)
		}

		for key, add := range built {
			output[key] = mergeSignatures(output[key], add)
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

	return output, flattenedSources, nil
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

func signatureKey(sig analysis.FunctionSignature) string {
	parts := []string{sig.Name}
	parts = append(parts, sig.Arguments...)
	return strings.Join(parts, "\x1f")
}

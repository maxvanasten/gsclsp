package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	outPath := flag.String("out", "analysis/stdlib_signatures.json", "Output json path")
	flag.Parse()

	if *mpRoot == "" || *zmRoot == "" {
		fmt.Fprintln(os.Stderr, "mp-root and zm-root are required")
		os.Exit(1)
	}

	mpMap, err := buildSignatureMap(*mpRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed building mp signatures: %v\n", err)
		os.Exit(1)
	}

	zmMap, err := buildSignatureMap(*zmRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed building zm signatures: %v\n", err)
		os.Exit(1)
	}

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

func buildSignatureMap(root string) (map[string][]analysis.FunctionSignature, error) {
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

		key := normalizeKey(rel)
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

		output[key] = signatures
		return nil
	})
	if err != nil {
		return nil, err
	}

	return output, nil
}

func normalizeKey(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimSuffix(path, ".gsc")
	path = strings.TrimPrefix(path, "/")
	path = strings.ToLower(path)
	return strings.TrimSpace(path)
}

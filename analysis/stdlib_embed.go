package analysis

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed stdlib_signatures.json
var stdlibSignaturesJSON embed.FS

var (
	stdlibOnce sync.Once
	stdlibData map[string]map[string][]FunctionSignature
	stdlibErr  error
)

func StdlibSignatures() (map[string]map[string][]FunctionSignature, error) {
	stdlibOnce.Do(func() {
		data, err := stdlibSignaturesJSON.ReadFile("stdlib_signatures.json")
		if err != nil {
			stdlibErr = fmt.Errorf("read stdlib signatures: %w", err)
			return
		}

		var parsed map[string]map[string][]FunctionSignature
		if err := json.Unmarshal(data, &parsed); err != nil {
			stdlibErr = fmt.Errorf("parse stdlib signatures: %w", err)
			return
		}
		stdlibData = parsed
	})

	return stdlibData, stdlibErr
}

func normalizeIncludePathBase(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, ".gsc")
	path = strings.TrimPrefix(path, "/")
	return path
}

func normalizeIncludeKey(path string) string {
	return strings.ToLower(normalizeIncludePathBase(path))
}

func guessStdlibGroup(uri string) string {
	uri = strings.ToLower(uri)
	if strings.Contains(uri, "/zm/") || strings.Contains(uri, "\\zm\\") {
		return "zm"
	}
	if strings.Contains(uri, "/mp/") || strings.Contains(uri, "\\mp\\") {
		return "mp"
	}
	return ""
}

func signatureKey(sig FunctionSignature) string {
	key := strings.Builder{}
	key.WriteString(sig.Name)
	key.WriteString("\x1f")
	key.WriteString(strings.Join(sig.Arguments, "\x1f"))
	return key.String()
}

func mergeSignatures(base []FunctionSignature, add []FunctionSignature) []FunctionSignature {
	if len(add) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(add))
	for _, sig := range base {
		seen[signatureKey(sig)] = struct{}{}
	}
	for _, sig := range add {
		key := signatureKey(sig)
		if _, ok := seen[key]; ok {
			continue
		}
		base = append(base, sig)
		seen[key] = struct{}{}
	}
	return base
}

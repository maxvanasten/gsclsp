package analysis

import (
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gscp/parser"
)

type SignatureResolver func(name string) (FunctionSignature, bool)

func findSignatureByName(signatures []FunctionSignature, name string) (FunctionSignature, bool) {
	if name == "" {
		return FunctionSignature{}, false
	}
	needle := strings.ToLower(name)
	for _, sig := range signatures {
		if strings.ToLower(sig.Name) == needle {
			return sig, true
		}
	}
	return FunctionSignature{}, false
}

func splitQualifiedName(name string) (string, string, bool) {
	idx := strings.LastIndex(name, "::")
	if idx <= 0 || idx+2 >= len(name) {
		return "", "", false
	}
	qualifier := strings.TrimSpace(name[:idx])
	funcName := strings.TrimSpace(name[idx+2:])
	if qualifier == "" || funcName == "" {
		return "", "", false
	}
	return qualifier, funcName, true
}

func resolveQualifiedSignature(stdlib map[string]map[string][]FunctionSignature, uri, name string) (FunctionSignature, bool) {
	qualifier, funcName, ok := splitQualifiedName(name)
	if !ok {
		return FunctionSignature{}, false
	}
	key := normalizeIncludeKey(qualifier)
	if key == "" {
		return FunctionSignature{}, false
	}

	stdlibGroup := guessStdlibGroup(uri)
	if stdlib != nil {
		if stdlibGroup != "" {
			if sigs, ok := stdlib[stdlibGroup][key]; ok {
				if sig, ok := findSignatureByName(sigs, funcName); ok {
					return sig, true
				}
			}
		}
		if sigs, ok := stdlib["mp"][key]; ok {
			if sig, ok := findSignatureByName(sigs, funcName); ok {
				return sig, true
			}
		}
		if sigs, ok := stdlib["zm"][key]; ok {
			if sig, ok := findSignatureByName(sigs, funcName); ok {
				return sig, true
			}
		}
	}

	return FunctionSignature{}, false
}

func resolveSignatureForName(name, uri string, local []FunctionSignature, stdlib map[string]map[string][]FunctionSignature) (FunctionSignature, bool) {
	if name == "" {
		return FunctionSignature{}, false
	}

	if sig, ok := findSignatureByName(local, name); ok {
		return sig, true
	}

	if _, funcName, ok := splitQualifiedName(name); ok {
		if sig, ok := findSignatureByName(local, funcName); ok {
			return sig, true
		}
		return resolveQualifiedSignature(stdlib, uri, name)
	}

	return FunctionSignature{}, false
}

func findFunctionCallNameAtPosition(nodes []parser.Node, position lsp.Position) string {
	for _, n := range nodes {
		if n.Type == "function_call" {
			line := n.Line - 1
			col := n.Col - 1
			if line == position.Line && col >= 0 {
				name := n.Data.FunctionName
				if n.Data.Path != "" {
					name = n.Data.Path + "::" + n.Data.FunctionName
				}
				end := col + len(name)
				if position.Character >= col && position.Character <= end {
					return name
				}
			}
		}
		if len(n.Children) > 0 {
			if name := findFunctionCallNameAtPosition(n.Children, position); name != "" {
				return name
			}
		}
	}

	return ""
}

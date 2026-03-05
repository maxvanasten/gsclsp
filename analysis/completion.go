package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/maxvanasten/gsclsp/lsp"
)

const maxCompletionItems = 200

type completionContextMode int

const (
	completionModeDefault completionContextMode = iota
	completionModeIncludePath
	completionModeQualifiedFunction
	completionModeQualifiedPath
)

type completionContext struct {
	Mode            completionContextMode
	Prefix          string
	PathPrefix      string
	Qualifier       string
	QualifiedPrefix string
}

func (s *State) Completion(id int, uri string, position lsp.Position) lsp.CompletionResponse {
	ctx := detectCompletionContext(s.Documents[uri], position)
	stdlib := s.loadStdlib()
	items := s.completionItemsForContext(uri, ctx, stdlib, maxCompletionItems)

	return lsp.CompletionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.CompletionList{
			IsIncomplete: len(items) == maxCompletionItems,
			Items:        items,
		},
	}
}

func (s *State) completionItemsForContext(uri string, ctx completionContext, stdlib map[string]map[string][]FunctionSignature, limit int) []lsp.CompletionItem {
	items := []lsp.CompletionItem{}

	switch ctx.Mode {
	case completionModeIncludePath:
		items = mergeCompletionItems(items, completionPathItems(collectIncludePathCandidates(uri, stdlib), ctx.PathPrefix))
	case completionModeQualifiedFunction:
		items = mergeCompletionItems(items, completionFunctionItems(resolveQualifiedCompletionSignatures(s, uri, ctx.Qualifier, stdlib), ctx.QualifiedPrefix))
	case completionModeQualifiedPath:
		items = mergeCompletionItems(items, completionPathItems(collectIncludePathCandidates(uri, stdlib), ctx.PathPrefix))
	default:
		items = mergeCompletionItems(items, completionFunctionItems(s.Signatures[uri], ctx.Prefix))
		items = mergeCompletionItems(items, completionKeywordItems(ctx.Prefix))
	}

	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

func completionFunctionItems(signatures []FunctionSignature, prefix string) []lsp.CompletionItem {
	if len(signatures) == 0 {
		return []lsp.CompletionItem{}
	}

	lowerPrefix := strings.ToLower(prefix)
	items := make([]lsp.CompletionItem, 0, len(signatures))
	for _, sig := range signatures {
		name := strings.TrimSpace(sig.Name)
		if name == "" {
			continue
		}
		if lowerPrefix != "" && !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			continue
		}

		items = append(items, lsp.CompletionItem{
			Label:            name,
			Kind:             lsp.CompletionItemKindFunction,
			Detail:           signatureDetail(sig),
			InsertText:       functionCompletionSnippet(sig),
			InsertTextFormat: lsp.InsertTextFormatSnippet,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		iLower := strings.ToLower(items[i].Label)
		jLower := strings.ToLower(items[j].Label)
		if iLower == jLower {
			return items[i].Label < items[j].Label
		}
		return iLower < jLower
	})

	return items
}

func completionKeywordItems(prefix string) []lsp.CompletionItem {
	lowerPrefix := strings.ToLower(prefix)
	items := make([]lsp.CompletionItem, 0, len(keywordTokens))
	for keyword := range keywordTokens {
		if keyword == "#include" {
			continue
		}
		if lowerPrefix != "" && !strings.HasPrefix(strings.ToLower(keyword), lowerPrefix) {
			continue
		}
		items = append(items, lsp.CompletionItem{
			Label:      keyword,
			Kind:       lsp.CompletionItemKindKeyword,
			InsertText: keyword,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		iLower := strings.ToLower(items[i].Label)
		jLower := strings.ToLower(items[j].Label)
		if iLower == jLower {
			return items[i].Label < items[j].Label
		}
		return iLower < jLower
	})

	return items
}

func completionPathItems(candidates []string, prefix string) []lsp.CompletionItem {
	prefixKey := normalizeIncludeKey(prefix)
	items := make([]lsp.CompletionItem, 0, len(candidates))
	for _, candidate := range candidates {
		if prefixKey != "" && !strings.HasPrefix(normalizeIncludeKey(candidate), prefixKey) {
			continue
		}
		pathLabel := slashPathToGsc(candidate)
		items = append(items, lsp.CompletionItem{
			Label:      pathLabel,
			Kind:       lsp.CompletionItemKindModule,
			InsertText: pathLabel,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		iLower := strings.ToLower(items[i].Label)
		jLower := strings.ToLower(items[j].Label)
		if iLower == jLower {
			return items[i].Label < items[j].Label
		}
		return iLower < jLower
	})

	return items
}

func resolveQualifiedCompletionSignatures(s *State, uri, qualifier string, stdlib map[string]map[string][]FunctionSignature) []FunctionSignature {
	key := normalizeIncludeKey(qualifier)
	if key == "" {
		return nil
	}

	if sigs, ok := lookupStdlibSignatures(stdlib, guessStdlibGroup(uri), key); ok {
		return sigs
	}

	resolvedPath, ok := resolveIncludePath(uri, qualifier)
	if !ok {
		return nil
	}
	entry, err := s.getParsedInclude(resolvedPath)
	if err != nil {
		return nil
	}

	return entry.Signatures
}

func collectIncludePathCandidates(uri string, stdlib map[string]map[string][]FunctionSignature) []string {
	seen := map[string]struct{}{}
	candidates := []string{}

	addCandidate := func(path string) {
		key := normalizeIncludeKey(path)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, key)
	}

	preferredGroup := guessStdlibGroup(uri)
	orderedGroups := []string{}
	if preferredGroup != "" {
		orderedGroups = append(orderedGroups, preferredGroup)
	}
	if preferredGroup != "mp" {
		orderedGroups = append(orderedGroups, "mp")
	}
	if preferredGroup != "zm" {
		orderedGroups = append(orderedGroups, "zm")
	}

	for _, group := range orderedGroups {
		for key := range stdlib[group] {
			addCandidate(key)
		}
	}

	docPath := uriToPath(uri)
	if docPath != "" {
		rootDir := filepath.Dir(docPath)
		_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".gsc") {
				return nil
			}
			rel, relErr := filepath.Rel(rootDir, path)
			if relErr != nil {
				return nil
			}
			addCandidate(filepath.ToSlash(rel))
			return nil
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i] < candidates[j]
	})

	return candidates
}

func mergeCompletionItems(base, add []lsp.CompletionItem) []lsp.CompletionItem {
	if len(add) == 0 {
		return base
	}

	itemIndex := make(map[string]int, len(base)+len(add))
	for i := range base {
		itemIndex[strings.ToLower(base[i].Label)] = i
	}

	for _, item := range add {
		key := strings.ToLower(item.Label)
		if idx, ok := itemIndex[key]; ok {
			if completionKindPriority(item.Kind) > completionKindPriority(base[idx].Kind) {
				base[idx] = item
			}
			continue
		}
		itemIndex[key] = len(base)
		base = append(base, item)
	}

	sort.Slice(base, func(i, j int) bool {
		iLower := strings.ToLower(base[i].Label)
		jLower := strings.ToLower(base[j].Label)
		if iLower == jLower {
			return base[i].Label < base[j].Label
		}
		return iLower < jLower
	})

	return base
}

func completionKindPriority(kind int) int {
	switch kind {
	case lsp.CompletionItemKindFunction:
		return 3
	case lsp.CompletionItemKindModule:
		return 2
	case lsp.CompletionItemKindKeyword:
		return 1
	default:
		return 0
	}
}

func detectCompletionContext(doc string, position lsp.Position) completionContext {
	linePrefix := linePrefixAtPosition(doc, position)
	if ok, includePrefix := includePathPrefixFromLine(linePrefix); ok {
		return completionContext{Mode: completionModeIncludePath, PathPrefix: includePrefix}
	}

	token := completionTokenAtLineEnd(linePrefix)
	if qualifier, qualifiedPrefix, ok := splitQualifiedPrefix(token); ok {
		return completionContext{Mode: completionModeQualifiedFunction, Qualifier: qualifier, QualifiedPrefix: qualifiedPrefix}
	}

	if strings.ContainsAny(token, `\\/`) {
		return completionContext{Mode: completionModeQualifiedPath, PathPrefix: token}
	}

	return completionContext{Mode: completionModeDefault, Prefix: completionPrefixAtPosition(doc, position)}
}

func linePrefixAtPosition(doc string, position lsp.Position) string {
	if doc == "" || position.Line < 0 {
		return ""
	}

	lines := strings.Split(doc, "\n")
	if position.Line >= len(lines) {
		return ""
	}
	line := lines[position.Line]
	if position.Character < 0 {
		return ""
	}
	if position.Character > len(line) {
		position.Character = len(line)
	}

	return line[:position.Character]
}

func includePathPrefixFromLine(linePrefix string) (bool, string) {
	lower := strings.ToLower(linePrefix)
	idx := strings.LastIndex(lower, "#include")
	if idx < 0 {
		return false, ""
	}
	tail := linePrefix[idx+len("#include"):]
	if strings.Contains(tail, ";") {
		return false, ""
	}

	return true, strings.TrimLeft(tail, " \t")
}

func completionTokenAtLineEnd(linePrefix string) string {
	if linePrefix == "" {
		return ""
	}

	start := len(linePrefix)
	for start > 0 {
		if !isCompletionTokenByte(linePrefix[start-1]) {
			break
		}
		start--
	}

	return linePrefix[start:]
}

func splitQualifiedPrefix(token string) (string, string, bool) {
	idx := strings.LastIndex(token, "::")
	if idx <= 0 {
		return "", "", false
	}

	qualifier := strings.TrimSpace(token[:idx])
	prefix := strings.TrimSpace(token[idx+2:])
	if qualifier == "" {
		return "", "", false
	}

	return qualifier, prefix, true
}

func signatureDetail(sig FunctionSignature) string {
	if len(sig.Arguments) == 0 {
		return sig.Name + "()"
	}
	return sig.Name + "(" + strings.Join(sig.Arguments, ", ") + ")"
}

func functionCompletionSnippet(sig FunctionSignature) string {
	if len(sig.Arguments) == 0 {
		return sig.Name + "()"
	}

	parts := make([]string, 0, len(sig.Arguments))
	for i, arg := range sig.Arguments {
		name := sanitizeSnippetPlaceholder(arg)
		if name == "" {
			name = "arg"
		}
		parts = append(parts, "${"+strconv.Itoa(i+1)+":"+name+"}")
	}
	return sig.Name + "(" + strings.Join(parts, ", ") + ")"
}

func sanitizeSnippetPlaceholder(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "")
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, "}", "")
	return value
}

func completionPrefixAtPosition(doc string, position lsp.Position) string {
	if doc == "" || position.Line < 0 || position.Character < 0 {
		return ""
	}

	lines := strings.Split(doc, "\n")
	if position.Line >= len(lines) {
		return ""
	}

	line := lines[position.Line]
	if position.Character > len(line) {
		position.Character = len(line)
	}

	start := position.Character
	for start > 0 {
		r := line[start-1]
		if !isIdentifierByte(r) {
			break
		}
		start--
	}

	return line[start:position.Character]
}

func isCompletionTokenByte(b byte) bool {
	if isIdentifierByte(b) {
		return true
	}

	switch b {
	case '\\', '/', ':', '.':
		return true
	default:
		return false
	}
}

func isIdentifierByte(b byte) bool {
	if b >= 'a' && b <= 'z' {
		return true
	}
	if b >= 'A' && b <= 'Z' {
		return true
	}
	if b >= '0' && b <= '9' {
		return true
	}
	return b == '_'
}

func slashPathToGsc(path string) string {
	return strings.ReplaceAll(path, "/", "\\")
}

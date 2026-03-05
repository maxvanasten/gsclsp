package lsp

import "testing"

func TestInitializeResponseAdvertisesCompletionProvider(t *testing.T) {
	resp := NewInitializeResponse(1)
	if !resp.Result.Capabilities.DocumentFormattingProvider {
		t.Fatal("expected documentFormattingProvider to be true")
	}
	foundComment := false
	for _, tokenType := range resp.Result.Capabilities.SemanticTokensProvider.Legend.TokenTypes {
		if tokenType == "comment" {
			foundComment = true
			break
		}
	}
	if !foundComment {
		t.Fatal("expected semantic token legend to include comment")
	}
	provider := resp.Result.Capabilities.CompletionProvider
	if provider == nil {
		t.Fatal("expected completion provider")
	}
	if provider.ResolveProvider {
		t.Fatal("expected resolveProvider to be false")
	}
	if len(provider.TriggerCharacters) == 0 {
		t.Fatal("expected trigger characters")
	}
}

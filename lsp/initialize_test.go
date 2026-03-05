package lsp

import "testing"

func TestInitializeResponseAdvertisesCompletionProvider(t *testing.T) {
	resp := NewInitializeResponse(1)
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

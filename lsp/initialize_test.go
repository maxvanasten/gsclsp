package lsp

import "testing"

func TestInitializeResponseAdvertisesCompletionProvider(t *testing.T) {
	resp := NewInitializeResponse(1)
	if !resp.Result.Capabilities.DocumentFormattingProvider {
		t.Fatal("expected documentFormattingProvider to be true")
	}
	if len(resp.Result.Capabilities.CodeActionProvider.CodeActionKinds) == 0 {
		t.Fatal("expected codeActionProvider to advertise kinds")
	}
	foundSource := false
	foundQuickfix := false
	for _, kind := range resp.Result.Capabilities.CodeActionProvider.CodeActionKinds {
		if kind == CodeActionKindSource {
			foundSource = true
		}
		if kind == CodeActionKindQuickFix {
			foundQuickfix = true
		}
	}
	if !foundSource || !foundQuickfix {
		t.Fatalf("expected source and quickfix codeAction kinds, got %v", resp.Result.Capabilities.CodeActionProvider.CodeActionKinds)
	}
	if len(resp.Result.Capabilities.ExecuteCommandProvider.Commands) == 0 {
		t.Fatal("expected executeCommandProvider commands")
	}
	if resp.Result.Capabilities.ExecuteCommandProvider.Commands[0] != "gsclsp.bundleMod" {
		t.Fatalf("unexpected execute command: %q", resp.Result.Capabilities.ExecuteCommandProvider.Commands[0])
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

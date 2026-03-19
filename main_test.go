package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxvanasten/gsclsp/analysis"
	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gsclsp/rpc"
)

func TestHandleMessageDefinitionMethodCanonical(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	request := lsp.DefinitionRequest{
		Request: lsp.Request{RPC: "2.0", ID: 1, Method: "textDocument/definition"},
		Params: lsp.DefinitionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
				Position:     lsp.Position{Line: 0, Character: 0},
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textDocument/definition", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.DefinitionResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 1 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
}

func TestHandleMessageCodeActionMethod(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	request := lsp.CodeActionRequest{
		Request: lsp.Request{RPC: "2.0", ID: 5, Method: "textDocument/codeAction"},
		Params: lsp.CodeActionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
			Range:        lsp.Range{},
			Context:      lsp.CodeActionContext{},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textDocument/codeAction", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.CodeActionResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 5 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one code action, got %d", len(response.Result))
	}
	action := response.Result[0]
	if action.Kind != lsp.CodeActionKindQuickFix {
		t.Fatalf("expected quickfix action kind, got %q", action.Kind)
	}
	if action.Command == nil || action.Command.Command != "gsclsp.bundleMod" {
		t.Fatalf("unexpected code action command: %+v", action.Command)
	}
}

func TestHandleMessageCodeActionMethodQuickfixFilter(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	request := lsp.CodeActionRequest{
		Request: lsp.Request{RPC: "2.0", ID: 7, Method: "textDocument/codeAction"},
		Params: lsp.CodeActionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
			Range:        lsp.Range{},
			Context: lsp.CodeActionContext{
				Only: []string{lsp.CodeActionKindQuickFix},
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textDocument/codeAction", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.CodeActionResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 7 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one code action, got %d", len(response.Result))
	}
	if response.Result[0].Kind != lsp.CodeActionKindQuickFix {
		t.Fatalf("expected quickfix action kind, got %q", response.Result[0].Kind)
	}
}

func TestHandleMessageCodeActionMethodUnsupportedFilter(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	request := lsp.CodeActionRequest{
		Request: lsp.Request{RPC: "2.0", ID: 8, Method: "textDocument/codeAction"},
		Params: lsp.CodeActionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
			Range:        lsp.Range{},
			Context: lsp.CodeActionContext{
				Only: []string{"refactor"},
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textDocument/codeAction", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.CodeActionResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 8 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
	if len(response.Result) != 0 {
		t.Fatalf("expected zero code actions for unsupported filter, got %d", len(response.Result))
	}
}

func TestHandleMessageExecuteCommandBundleMod(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	root := t.TempDir()
	sourceRoot := filepath.Join(root, "zm_test")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.gsc"), []byte("main(){}"), 0o644); err != nil {
		t.Fatalf("write source script: %v", err)
	}

	uri := "file://" + filepath.ToSlash(filepath.Join(sourceRoot, "main.gsc"))
	request := lsp.ExecuteCommandRequest{
		Request: lsp.Request{RPC: "2.0", ID: 6, Method: "workspace/executeCommand"},
		Params: lsp.ExecuteCommandParams{
			Command:   "gsclsp.bundleMod",
			Arguments: []any{uri},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "workspace/executeCommand", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.ExecuteCommandResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 6 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
	result, ok := response.Result.(string)
	if !ok || !strings.Contains(result, "Bundled") {
		t.Fatalf("unexpected execute command result: %#v", response.Result)
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "zm_test", "scripts", "main.gsc")); err != nil {
		t.Fatalf("expected bundled script: %v", err)
	}
}

func TestHandleMessageDefinitionMethodLegacyCasing(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	request := lsp.DefinitionRequest{
		Request: lsp.Request{RPC: "2.0", ID: 2, Method: "textdocument/definition"},
		Params: lsp.DefinitionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
				Position:     lsp.Position{Line: 0, Character: 0},
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textdocument/definition", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.DefinitionResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 2 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
}

func TestHandleMessageCompletionMethod(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	state.Documents["file:///tmp/test.gsc"] = "main() { te }"
	state.Signatures["file:///tmp/test.gsc"] = []analysis.FunctionSignature{
		{Name: "test_fn", Arguments: []string{"arg"}},
	}

	request := lsp.CompletionRequest{
		Request: lsp.Request{RPC: "2.0", ID: 3, Method: "textDocument/completion"},
		Params: lsp.CompletionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
				Position:     lsp.Position{Line: 0, Character: 11},
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textDocument/completion", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.CompletionResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 3 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
	if len(response.Result.Items) != 1 || response.Result.Items[0].Label != "test_fn" {
		t.Fatalf("unexpected completion items: %+v", response.Result.Items)
	}
}

func TestHandleMessageDocumentFormattingMethod(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	logger := log.New(io.Discard, "", 0)
	var out bytes.Buffer

	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){wait 0.05;}"

	request := lsp.DocumentFormattingRequest{
		Request: lsp.Request{RPC: "2.0", ID: 4, Method: "textDocument/formatting"},
		Params: lsp.DocumentFormattingParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Options: lsp.FormattingOptions{
				TabSize:      2,
				InsertSpaces: true,
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	handleMessage(logger, &out, &state, "textDocument/formatting", contents)

	_, payload, err := rpc.DecodeMessage(out.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var response lsp.DocumentFormattingResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID == nil || *response.ID != 4 {
		t.Fatalf("unexpected response id: %v", response.ID)
	}
	if len(response.Result) != 1 {
		t.Fatalf("expected one text edit, got %d", len(response.Result))
	}
	if response.Result[0].NewText == "" {
		t.Fatal("expected formatted content")
	}
}

func TestHandleMessagePanicsAreRecovered(t *testing.T) {
	t.Helper()
	state := analysis.NewState()
	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "[test] ", 0)
	var out bytes.Buffer

	request := lsp.HoverRequest{
		Request: lsp.Request{RPC: "2.0", ID: 99, Method: "textDocument/hover"},
		Params: lsp.HoverParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{URI: "file:///tmp/test.gsc"},
				Position:     lsp.Position{Line: 0, Character: 0},
			},
		},
	}
	contents, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	// This should not panic even though the document doesn't exist
	// and may cause nil pointer dereference in some code paths
	handleMessage(logger, &out, &state, "textDocument/hover", contents)

	// Verify the function returned (didn't crash the test)
	// The response might be empty but that's okay - we just want to ensure no panic
	t.Log("handleMessage completed without panic")
}

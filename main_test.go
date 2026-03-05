package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
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

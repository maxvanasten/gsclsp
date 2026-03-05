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

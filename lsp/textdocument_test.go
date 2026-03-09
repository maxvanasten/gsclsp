package lsp

import (
	"encoding/json"
	"testing"
)

func TestDidOpenUsesLanguageIdFieldName(t *testing.T) {
	notification := DidOpenTextDocumentNotification{
		Notification: Notification{RPC: "2.0", Method: "textDocument/didOpen"},
		Params: DidOpenTextDocumentParams{
			TextDocument: TextDocumentItem{
				URI:        "file:///tmp/test.gsc",
				LanguageID: "gsc",
				Version:    1,
				Text:       "main() {}",
			},
		},
	}

	data, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	params, ok := payload["params"].(map[string]any)
	if !ok {
		t.Fatalf("params has unexpected type: %T", payload["params"])
	}
	textDocument, ok := params["textDocument"].(map[string]any)
	if !ok {
		t.Fatalf("textDocument has unexpected type: %T", params["textDocument"])
	}

	if _, exists := textDocument["languageId"]; !exists {
		t.Fatalf("expected languageId field in payload: %s", string(data))
	}
	if _, exists := textDocument["languageid"]; exists {
		t.Fatalf("unexpected lowercase languageid field in payload: %s", string(data))
	}
}

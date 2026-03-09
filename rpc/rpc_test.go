package rpc_test

import (
	"testing"

	"github.com/maxvanasten/gsclsp/rpc"
)

type EncodingExample struct {
	Testing bool
}

func TestEncode(t *testing.T) {
	expected := "Content-Length: 16\r\n\r\n{\"Testing\":true}"
	actual, err := rpc.EncodeMessage(EncodingExample{Testing: true})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if expected != actual {
		t.Fatalf("Expected: %s, Actual: %s", expected, actual)
	}
}

func TestDecode(t *testing.T) {
	incomingMessage := "Content-Length: 15\r\n\r\n{\"Method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err != nil {
		t.Fatal(err)
	}

	if len(content) != 15 {
		t.Fatalf("Expected: 15, got: %d\n", len(content))
	}

	if method != "hi" {
		t.Fatalf("Expected: hi, got: %s\n", method)
	}
}

func TestDecodeWithMultipleHeaders(t *testing.T) {
	incomingMessage := "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\nContent-Length: 15\r\n\r\n{\"method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(content) != 15 {
		t.Fatalf("Expected: 15, got: %d", len(content))
	}
	if method != "hi" {
		t.Fatalf("Expected: hi, got: %s", method)
	}
}

func TestDecodeWithoutContentLengthHeader(t *testing.T) {
	incomingMessage := "Content-Type: application/json\r\n\r\n{}"
	_, _, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err == nil {
		t.Fatal("expected decode to fail without Content-Length")
	}
}

func TestDecodeShortContent(t *testing.T) {
	incomingMessage := "Content-Length: 20\r\n\r\n{\"method\":\"hi\"}"
	_, _, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err == nil {
		t.Fatal("expected decode to fail for short content")
	}
}

func TestDecodeMixedCaseContentLengthHeader(t *testing.T) {
	incomingMessage := "content-length: 15\r\n\r\n{\"method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(content) != 15 {
		t.Fatalf("Expected: 15, got: %d", len(content))
	}
	if method != "hi" {
		t.Fatalf("Expected: hi, got: %s", method)
	}
}

func TestDecodeInvalidContentLengthValue(t *testing.T) {
	incomingMessage := "Content-Length: abc\r\n\r\n{\"method\":\"hi\"}"
	_, _, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err == nil {
		t.Fatal("expected decode to fail for invalid Content-Length value")
	}
}

package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/maxvanasten/gsclsp/analysis"
	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gsclsp/rpc"
)

func main() {
	logger := getLogger()
	logger.Println("Started")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	scanner.Split(rpc.Split)

	state := analysis.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Got an error: %s", err)
			continue
		}
		handleMessage(logger, writer, &state, method, contents)
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("Scanner error: %s", err)
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, state *analysis.State, method string, contents []byte) {
	logger.Printf("Received message with method: %s", method)

	switch method {
	case "initialize":
		var request lsp.InitializeRequest
		decodeRequest(logger, contents, &request, "Json unmarshaling err")

		logger.Printf("Connected to: %s %s", request.Params.ClientInfo.Name, request.Params.ClientInfo.Version)

		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(writer, msg)

		logger.Print("Sent the reply")
	case "textDocument/didOpen":
		var request lsp.DidOpenTextDocumentNotification
		decodeRequest(logger, contents, &request, "Json unmarsheling err")

		logger.Printf("Opened: %s", request.Params.TextDocument.URI)

		state.OpenDocument(request.Params.TextDocument.URI, request.Params.TextDocument.Text)
		publishDiagnostics(writer, request.Params.TextDocument.URI, state.Diagnostics[request.Params.TextDocument.URI])
	case "textDocument/didChange":
		var request lsp.TextDocumentDidChangeNotification
		if !decodeRequest(logger, contents, &request, "textDocument/didChange") {
			return
		}

		logger.Printf("Changed: %s", request.Params.TextDocument.URI)
		for _, change := range request.Params.ContentChanges {
			state.UpdateDocument(request.Params.TextDocument.URI, change.Text)
		}
		publishDiagnostics(writer, request.Params.TextDocument.URI, state.Diagnostics[request.Params.TextDocument.URI])
	case "textDocument/hover":
		var request lsp.HoverRequest
		if !decodeRequest(logger, contents, &request, "textDocument/hover") {
			return
		}

		response := state.Hover(request.ID, request.Params.TextDocument.URI, request.Params.Position)
		writeResponse(writer, response)

	case "textDocument/definition", "textdocument/definition":
		var request lsp.DefinitionRequest
		if !decodeRequest(logger, contents, &request, "textDocument/definition") {
			return
		}

		response := state.Definition(request.ID, request.Params.TextDocument.URI, request.Params.Position)
		writeResponse(writer, response)
	case "textDocument/completion":
		var request lsp.CompletionRequest
		if !decodeRequest(logger, contents, &request, "textDocument/completion") {
			return
		}

		response := state.Completion(request.ID, request.Params.TextDocument.URI, request.Params.Position)
		writeResponse(writer, response)
	case "textDocument/semanticTokens/full":
		var request lsp.SemanticTokensRequest
		if !decodeRequest(logger, contents, &request, "textDocument/semanticTokens/full") {
			return
		}

		response := state.SemanticTokens(request.ID, request.Params.TextDocument.URI)
		writeResponse(writer, response)
		logger.Printf("semantic_tokens: %v", response.Result.Data)
	case "textDocument/inlayHint":
		var request lsp.InlayHintRequest
		if !decodeRequest(logger, contents, &request, "textDocument/inlayHint") {
			return
		}

		response := state.InlayHints(request.ID, request.Params.TextDocument.URI)
		writeResponse(writer, response)
		logger.Printf("inlay_hints: %v", response.Result)
	}
}

func decodeRequest(logger *log.Logger, contents []byte, target any, errPrefix string) bool {
	if err := json.Unmarshal(contents, target); err != nil {
		logger.Printf("%s: %s", errPrefix, err)
		return false
	}
	return true
}

func writeResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))
}

func publishDiagnostics(writer io.Writer, uri string, diagnostics []lsp.Diagnostic) {
	if diagnostics == nil {
		diagnostics = []lsp.Diagnostic{}
	}
	msg := lsp.PublishDiagnosticsNotification{
		Notification: lsp.Notification{
			RPC:    "2.0",
			Method: "textDocument/publishDiagnostics",
		},
		Params: lsp.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		},
	}
	writeResponse(writer, msg)
}

func getLogger() *log.Logger {
	return log.New(os.Stderr, "[gsclsp] ", log.Ldate|log.Ltime|log.Lshortfile)
}

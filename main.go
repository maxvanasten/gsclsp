package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/maxvanasten/gsclsp/analysis"
	"github.com/maxvanasten/gsclsp/lsp"
	"github.com/maxvanasten/gsclsp/rpc"
)

var diagDebouncer *diagnosticDebouncer

type diagnosticDebouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

func newDiagnosticDebouncer() *diagnosticDebouncer {
	return &diagnosticDebouncer{
		timers: make(map[string]*time.Timer),
	}
}

func (d *diagnosticDebouncer) debounce(uri string, delay time.Duration, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.timers[uri]; ok {
		t.Stop()
	}
	d.timers[uri] = time.AfterFunc(delay, fn)
}

func main() {
	logger := getLogger()
	logger.Println("Started")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	scanner.Split(rpc.Split)

	state := analysis.NewState()
	diagDebouncer = newDiagnosticDebouncer()
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			if err := state.Close(); err != nil {
				logger.Printf("cleanup error: %v", err)
			}
		})
	}
	defer shutdown()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		logger.Print("Received shutdown signal")
		shutdown()
		os.Exit(0)
	}()

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
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("PANIC in handleMessage for method %s: %v", method, r)
		}
	}()

	logger.Printf("Received message with method: %s", method)

	switch method {
	case "initialize":
		var request lsp.InitializeRequest
		decodeRequest(logger, contents, &request, "Json unmarshaling err")

		logger.Printf("Connected to: %s %s", request.Params.ClientInfo.Name, request.Params.ClientInfo.Version)

		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(logger, writer, msg)

		logger.Print("Sent the reply")
	case "initialized":
		var request lsp.InitializedNotification
		if !decodeRequest(logger, contents, &request, "initialized") {
			return
		}
		workspaceRoot := detectWorkspaceRootFromFirstDocument(state)
		if workspaceRoot != "" {
			state.SetWorkspaceFolders([]string{workspaceRoot})
			logger.Printf("Auto-detected workspace root: %s", workspaceRoot)
		}
	case "workspace/workspaceFolders":
		var request lsp.WorkspaceFoldersNotification
		if !decodeRequest(logger, contents, &request, "workspace/workspaceFolders") {
			return
		}
		folders := make([]string, len(request.Params.WorkspaceFolders))
		for i, wf := range request.Params.WorkspaceFolders {
			folders[i] = wf.URI
		}
		state.SetWorkspaceFolders(folders)
		logger.Printf("Set workspace folders: %v", folders)
	case "textDocument/didOpen":
		var request lsp.DidOpenTextDocumentNotification
		decodeRequest(logger, contents, &request, "Json unmarsheling err")

		logger.Printf("Opened: %s", request.Params.TextDocument.URI)

		if len(state.WorkspaceFolders()) == 0 {
			workspaceRoot := analysis.DetectWorkspaceRootFromDocument(request.Params.TextDocument.URI)
			if workspaceRoot != "" {
				state.SetWorkspaceFolders([]string{workspaceRoot})
				logger.Printf("Auto-detected workspace root from didOpen: %s", workspaceRoot)
			}
		}

		state.OpenDocument(request.Params.TextDocument.URI, request.Params.TextDocument.Text)
		publishDiagnostics(logger, writer, request.Params.TextDocument.URI, state.Diagnostics[request.Params.TextDocument.URI])
	case "textDocument/didChange":
		var request lsp.TextDocumentDidChangeNotification
		if !decodeRequest(logger, contents, &request, "textDocument/didChange") {
			return
		}

		logger.Printf("Changed: %s", request.Params.TextDocument.URI)
		for _, change := range request.Params.ContentChanges {
			state.ApplyIncrementalChange(request.Params.TextDocument.URI, change)
		}
		uri := request.Params.TextDocument.URI
		diagDebouncer.debounce(uri, 150*time.Millisecond, func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Printf("PANIC in diagnostic debounce for %s: %v", uri, r)
				}
			}()
			logger.Printf("Parsing: %s", uri)
			state.EnsureParsed(uri)
			diagnostics := state.Diagnostics[uri]
			logger.Printf("Publishing %d diagnostics for %s", len(diagnostics), uri)
			publishDiagnostics(logger, writer, uri, diagnostics)
		})
	case "textDocument/hover":
		var request lsp.HoverRequest
		if !decodeRequest(logger, contents, &request, "textDocument/hover") {
			return
		}

		response := state.Hover(request.ID, request.Params.TextDocument.URI, request.Params.Position)
		writeResponse(logger, writer, response)

	case "textDocument/definition", "textdocument/definition":
		var request lsp.DefinitionRequest
		if !decodeRequest(logger, contents, &request, "textDocument/definition") {
			return
		}

		response := state.Definition(request.ID, request.Params.TextDocument.URI, request.Params.Position)
		writeResponse(logger, writer, response)
	case "textDocument/completion":
		var request lsp.CompletionRequest
		if !decodeRequest(logger, contents, &request, "textDocument/completion") {
			return
		}

		response := state.Completion(request.ID, request.Params.TextDocument.URI, request.Params.Position)
		writeResponse(logger, writer, response)
	case "textDocument/semanticTokens/full":
		var request lsp.SemanticTokensRequest
		if !decodeRequest(logger, contents, &request, "textDocument/semanticTokens/full") {
			return
		}

		response := state.SemanticTokens(request.ID, request.Params.TextDocument.URI)
		writeResponse(logger, writer, response)
		logger.Printf("semantic_tokens: %d tokens", len(response.Result.Data)/5)
	case "textDocument/inlayHint":
		var request lsp.InlayHintRequest
		if !decodeRequest(logger, contents, &request, "textDocument/inlayHint") {
			return
		}

		response := state.InlayHints(request.ID, request.Params.TextDocument.URI)
		writeResponse(logger, writer, response)
		logger.Printf("inlay_hints: %d", len(response.Result))
		if os.Getenv("GSCLSP_DEBUG_INLAY") != "" {
			uri := request.Params.TextDocument.URI
			text := state.DocumentText(uri)
			lines := strings.Split(text, "\n")
			for _, hint := range response.Result {
				line := hint.Position.Line
				lineText := ""
				if line >= 0 && line < len(lines) {
					lineText = lines[line]
				}
				if lineText == "" {
					continue
				}
				if !strings.Contains(lineText, "add_quest(") {
					continue
				}
				logger.Printf("inlay_hint_debug label=%q line=%d col=%d line_text=%q", hint.Label, line, hint.Position.Character, lineText)
			}
		}
	case "textDocument/formatting":
		var request lsp.DocumentFormattingRequest
		if !decodeRequest(logger, contents, &request, "textDocument/formatting") {
			return
		}

		response := state.Formatting(request.ID, request.Params.TextDocument.URI, request.Params.Options)
		writeResponse(logger, writer, response)
	case "textDocument/codeAction":
		var request lsp.CodeActionRequest
		if !decodeRequest(logger, contents, &request, "textDocument/codeAction") {
			return
		}

		actionKind := lsp.CodeActionKindQuickFix
		if !includesRequestedCodeActionKind(request.Params.Context.Only, actionKind) {
			response := lsp.CodeActionResponse{
				Response: lsp.Response{RPC: "2.0", ID: &request.ID},
				Result:   []lsp.CodeAction{},
			}
			writeResponse(logger, writer, response)
			return
		}

		response := lsp.CodeActionResponse{
			Response: lsp.Response{RPC: "2.0", ID: &request.ID},
			Result: []lsp.CodeAction{
				{
					Title: "Bundle scripts into mod folder",
					Kind:  actionKind,
					Command: &lsp.Command{
						Title:     "Bundle scripts into mod folder",
						Command:   "gsclsp.bundleMod",
						Arguments: []any{request.Params.TextDocument.URI},
					},
				},
			},
		}
		writeResponse(logger, writer, response)
	case "workspace/executeCommand":
		var request lsp.ExecuteCommandRequest
		if !decodeRequest(logger, contents, &request, "workspace/executeCommand") {
			return
		}

		result := "unsupported command"
		switch request.Params.Command {
		case "gsclsp.bundleMod":
			uri, ok := commandURI(request.Params.Arguments)
			if !ok {
				result = "bundle failed: missing document uri argument"
				break
			}
			message, err := analysis.BundleModForURI(uri)
			if err != nil {
				result = "bundle failed: " + err.Error()
				break
			}
			result = message
		}

		response := lsp.ExecuteCommandResponse{
			Response: lsp.Response{RPC: "2.0", ID: &request.ID},
			Result:   result,
		}
		writeResponse(logger, writer, response)
	}
}

func commandURI(arguments []any) (string, bool) {
	if len(arguments) == 0 {
		return "", false
	}
	if value, ok := arguments[0].(string); ok && value != "" {
		return value, true
	}
	if value, ok := arguments[0].(map[string]any); ok {
		if uri, exists := value["uri"].(string); exists && uri != "" {
			return uri, true
		}
	}
	return "", false
}

func includesRequestedCodeActionKind(only []string, actionKind string) bool {
	if len(only) == 0 {
		return true
	}
	for _, requestedKind := range only {
		if requestedKind == actionKind || strings.HasPrefix(actionKind, requestedKind+".") {
			return true
		}
	}
	return false
}

func decodeRequest(logger *log.Logger, contents []byte, target any, errPrefix string) bool {
	if err := json.Unmarshal(contents, target); err != nil {
		logger.Printf("%s: %s", errPrefix, err)
		return false
	}
	return true
}

func writeResponse(logger *log.Logger, writer io.Writer, msg any) {
	reply, err := rpc.EncodeMessage(msg)
	if err != nil {
		logger.Printf("encode response error: %v", err)
		return
	}
	if _, err := writer.Write([]byte(reply)); err != nil {
		logger.Printf("write response error: %v", err)
	}
}

func publishDiagnostics(logger *log.Logger, writer io.Writer, uri string, diagnostics []lsp.Diagnostic) {
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
	writeResponse(logger, writer, msg)
}

func getLogger() *log.Logger {
	return log.New(os.Stderr, "[gsclsp] ", log.Ldate|log.Ltime|log.Lshortfile)
}

func detectWorkspaceRootFromFirstDocument(state *analysis.State) string {
	for uri := range state.Documents {
		if root := analysis.DetectWorkspaceRootFromDocument(uri); root != "" {
			return root
		}
	}
	return ""
}

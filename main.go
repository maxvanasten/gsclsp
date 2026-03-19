package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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

const (
	watchdogInterval   = 30 * time.Second
	watchdogTimeout    = 120 * time.Second // Log warning if no message received for 2 minutes
	stdoutWriteTimeout = 10 * time.Second
)

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

type watchdogTimer struct {
	logger    *log.Logger
	interval  time.Duration
	timeout   time.Duration
	ticker    *time.Ticker
	stopCh    chan struct{}
	isRunning bool
}

func (w *watchdogTimer) start(getLastMessage func() time.Time) {
	w.ticker = time.NewTicker(w.interval)
	w.stopCh = make(chan struct{})
	w.isRunning = true

	go func() {
		for {
			select {
			case <-w.ticker.C:
				if !w.isRunning {
					return
				}
				lastMsg := getLastMessage()
				if time.Since(lastMsg) > w.timeout {
					w.logger.Printf("WATCHDOG WARNING: No message received for %v, server may be hung", time.Since(lastMsg))
				}
			case <-w.stopCh:
				return
			}
		}
	}()
}

func (w *watchdogTimer) stop() {
	if !w.isRunning {
		return
	}
	w.isRunning = false
	w.ticker.Stop()
	close(w.stopCh)
}

type timeoutWriter struct {
	w       io.Writer
	timeout time.Duration
	logger  *log.Logger
}

func (tw *timeoutWriter) Write(p []byte) (n int, err error) {
	done := make(chan struct{})
	var writeErr error
	var bytesWritten int

	go func() {
		bytesWritten, writeErr = tw.w.Write(p)
		close(done)
	}()

	select {
	case <-done:
		return bytesWritten, writeErr
	case <-time.After(tw.timeout):
		tw.logger.Printf("Write timeout after %v", tw.timeout)
		return 0, fmt.Errorf("write timeout after %v", tw.timeout)
	}
}

func main() {
	logger := getLogger()
	logger.Println("Started")

	// Initialize watchdog to detect hangs
	lastMessageTime := time.Now()
	var lastMsgMu sync.Mutex

	watchdog := &watchdogTimer{
		logger:    logger,
		interval:  watchdogInterval,
		timeout:   watchdogTimeout,
		isRunning: true,
	}

	updateLastMessageTime := func() {
		lastMsgMu.Lock()
		lastMessageTime = time.Now()
		lastMsgMu.Unlock()
	}

	getLastMessageTime := func() time.Time {
		lastMsgMu.Lock()
		defer lastMsgMu.Unlock()
		return lastMessageTime
	}

	watchdog.start(getLastMessageTime)
	defer watchdog.stop()

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

	writer := &timeoutWriter{
		w:       os.Stdout,
		timeout: stdoutWriteTimeout,
		logger:  logger,
	}

	for scanner.Scan() {
		updateLastMessageTime()
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
		// Don't exit immediately - give watchdog a chance to detect the issue
		time.Sleep(100 * time.Millisecond)
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

		// Get code actions for missing includes
		includeActions := getIncludeCodeActions(logger, state, request.Params.TextDocument.URI, actionKind)

		actions := []lsp.CodeAction{
			{
				Title: "Bundle scripts into mod folder",
				Kind:  actionKind,
				Command: &lsp.Command{
					Title:     "Bundle scripts into mod folder",
					Command:   "gsclsp.bundleMod",
					Arguments: []any{request.Params.TextDocument.URI},
				},
			},
		}
		actions = append(actions, includeActions...)

		response := lsp.CodeActionResponse{
			Response: lsp.Response{RPC: "2.0", ID: &request.ID},
			Result:   actions,
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
		case "gsclsp.addInclude":
			if len(request.Params.Arguments) < 2 {
				result = "add include failed: missing arguments"
				break
			}
			uri, ok1 := request.Params.Arguments[0].(string)
			includePath, ok2 := request.Params.Arguments[1].(string)
			if !ok1 || !ok2 {
				result = "add include failed: invalid arguments"
				break
			}
			funcName := ""
			if len(request.Params.Arguments) > 2 {
				funcName, _ = request.Params.Arguments[2].(string)
			}
			// Add the include to the document
			text := state.DocumentText(uri)
			includeLine := fmt.Sprintf("#include %s;", includePath)
			// Check if already included
			if strings.Contains(text, includeLine) {
				result = fmt.Sprintf("#include for '%s' already exists", funcName)
				break
			}
			// Add at the beginning
			newText := includeLine + "\n" + text
			state.UpdateDocument(uri, newText)
			result = fmt.Sprintf("Added #include for '%s'", funcName)
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

func getIncludeCodeActions(logger *log.Logger, state *analysis.State, uri string, actionKind string) []lsp.CodeAction {
	missing := state.GetMissingFunctionIncludes(uri)
	if len(missing) == 0 {
		return nil
	}

	var actions []lsp.CodeAction
	for funcName, sources := range missing {
		if len(sources) == 0 {
			continue
		}

		// Use the first (most likely) source
		includePath := sources[0]
		includePath = strings.ReplaceAll(includePath, "/", "\\")

		title := fmt.Sprintf("Add #include for '%s' (%s)", funcName, includePath)
		if len(sources) > 1 {
			title = fmt.Sprintf("Add #include for '%s' (%s +%d more)", funcName, includePath, len(sources)-1)
		}

		action := lsp.CodeAction{
			Title: title,
			Kind:  actionKind,
			Command: &lsp.Command{
				Title:     title,
				Command:   "gsclsp.addInclude",
				Arguments: []any{uri, includePath, funcName},
			},
		}
		actions = append(actions, action)
	}

	return actions
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

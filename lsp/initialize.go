package lsp

type InitializeRequest struct {
	Request
	Params InitializeRequestParams `json:"params"`
}

type InitializeRequestParams struct {
	ClientInfo *ClientInfo `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResponse struct {
	Response
	Result InitializeResult `json:"result"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	TextDocumentSync           int                    `json:"textDocumentSync"`
	HoverProvider              bool                   `json:"hoverProvider"`
	DefinitionProvider         bool                   `json:"definitionProvider"`
	DocumentFormattingProvider bool                   `json:"documentFormattingProvider"`
	CodeActionProvider         CodeActionOptions      `json:"codeActionProvider"`
	ExecuteCommandProvider     ExecuteCommandOptions  `json:"executeCommandProvider"`
	CompletionProvider         *CompletionOptions     `json:"completionProvider,omitempty"`
	SemanticTokensProvider     SemanticTokensProvider `json:"semanticTokensProvider"`
	InlayHintProvider          bool                   `json:"inlayHintProvider"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewInitializeResponse(id int) InitializeResponse {
	return InitializeResponse{
		Response: Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync:           2,
				HoverProvider:              true,
				DefinitionProvider:         true,
				DocumentFormattingProvider: true,
				CodeActionProvider: CodeActionOptions{
					CodeActionKinds: []string{CodeActionKindQuickFix},
				},
				ExecuteCommandProvider: ExecuteCommandOptions{
					Commands: []string{"gsclsp.bundleMod"},
				},
				CompletionProvider: &CompletionOptions{
					ResolveProvider:   false,
					TriggerCharacters: []string{"\\", ":", "."},
				},
				SemanticTokensProvider: SemanticTokensProvider{
					Legend: SemanticTokensLegend{
						TokenTypes: []string{
							"variable",
							"keyword",
							"string",
							"number",
							"function",
							"property",
							"comment",
						},
						TokenModifiers: []string{},
					},
					Full:  true,
					Range: false,
				},
				InlayHintProvider: true,
			},
			ServerInfo: ServerInfo{
				Name:    "gsclsp",
				Version: "0.8.1",
			},
		},
	}
}

package lsp

type SemanticTokensProvider struct {
	Legend SemanticTokensLegend `json:"legend"`
	Full bool `json:"full"`
	Range bool `json:"range"`
}

type SemanticTokensLegend struct {
	TokenTypes []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type SemanticTokensRequest struct {
	Request
	Params SemanticTokensParams `json:"params"`
}

type SemanticTokensParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type SemanticTokensResult struct {
	Data []int `json:"data"`
}

type SemanticTokensResponse struct {
	Response
	Result SemanticTokensResult `json:"result"`
}

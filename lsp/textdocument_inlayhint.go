package lsp

type InlayHintRequest struct {
	Request
	Params InlayHintParams `json:"params"`
}

type InlayHintParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type InlayHintResponse struct {
	Response
	Result []InlayHint `json:"result"`
}

type InlayHint struct {
	Position Position `json:"position"`
	Label    string   `json:"label"`
}

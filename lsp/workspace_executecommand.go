package lsp

type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

type ExecuteCommandRequest struct {
	Request
	Params ExecuteCommandParams `json:"params"`
}

type ExecuteCommandParams struct {
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

type ExecuteCommandResponse struct {
	Response
	Result any `json:"result,omitempty"`
}

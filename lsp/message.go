package lsp

type Request struct {
	RPC    string `json:"jsonrpc"`
	ID     int    `json:"id"`
	Method string `json:"method"`
}

type Response struct {
	RPC string `json:"jsonrpc"`
	ID  *int   `json:"id,omitempty"`
}

type Notification struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
}

type InitializedNotification struct {
	Notification
	Params InitializedParams `json:"params"`
}

type InitializedParams struct {
}

type WorkspaceFoldersNotification struct {
	Notification
	Params WorkspaceFoldersParams `json:"params"`
}

type WorkspaceFoldersParams struct {
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

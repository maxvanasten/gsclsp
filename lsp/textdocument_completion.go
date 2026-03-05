package lsp

type CompletionRequest struct {
	Request
	Params CompletionParams `json:"params"`
}

type CompletionParams struct {
	TextDocumentPositionParams
}

type CompletionResponse struct {
	Response
	Result CompletionList `json:"result"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

type CompletionItem struct {
	Label            string `json:"label"`
	Kind             int    `json:"kind,omitempty"`
	Detail           string `json:"detail,omitempty"`
	InsertText       string `json:"insertText,omitempty"`
	InsertTextFormat int    `json:"insertTextFormat,omitempty"`
}

type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

const CompletionItemKindFunction = 3
const CompletionItemKindModule = 9
const CompletionItemKindKeyword = 14

const InsertTextFormatPlainText = 1
const InsertTextFormatSnippet = 2

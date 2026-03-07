package analysis

type StdlibDeclaration struct {
	Name        string   `json:"name"`
	Arguments   []string `json:"arguments"`
	Declaration string   `json:"declaration"`
}

package analysis

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed stdlib_declarations.json
var stdlibDeclarationsJSON embed.FS

var (
	stdlibDeclarationsOnce sync.Once
	stdlibDeclarationsData map[string]map[string][]StdlibDeclaration
	stdlibDeclarationsErr  error
)

func StdlibDeclarations() (map[string]map[string][]StdlibDeclaration, error) {
	stdlibDeclarationsOnce.Do(func() {
		data, err := stdlibDeclarationsJSON.ReadFile("stdlib_declarations.json")
		if err != nil {
			stdlibDeclarationsErr = fmt.Errorf("read stdlib declarations: %w", err)
			return
		}

		var parsed map[string]map[string][]StdlibDeclaration
		if err := json.Unmarshal(data, &parsed); err != nil {
			stdlibDeclarationsErr = fmt.Errorf("parse stdlib declarations: %w", err)
			return
		}
		stdlibDeclarationsData = parsed
	})

	return stdlibDeclarationsData, stdlibDeclarationsErr
}

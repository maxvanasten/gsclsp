package analysis

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed builtins_signatures.json
var builtinsSignaturesJSON embed.FS

var (
	builtinsOnce sync.Once
	builtinsData []FunctionSignature
	builtinsErr  error
)

func BuiltinsSignatures() ([]FunctionSignature, error) {
	builtinsOnce.Do(func() {
		data, err := builtinsSignaturesJSON.ReadFile("builtins_signatures.json")
		if err != nil {
			builtinsErr = fmt.Errorf("read builtins signatures: %w", err)
			return
		}

		var parsed []FunctionSignature
		if err := json.Unmarshal(data, &parsed); err != nil {
			builtinsErr = fmt.Errorf("parse builtins signatures: %w", err)
			return
		}
		builtinsData = parsed
	})

	return builtinsData, builtinsErr
}

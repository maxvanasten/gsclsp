package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGscpBenchmark(b *testing.B) {
	b.Helper()
	if _, err := exec.LookPath("gscp"); err != nil {
		b.Skipf("gscp is required for benchmark: %v", err)
	}
}

func setupBenchmarkState(b *testing.B) (*State, string, string, string) {
	b.Helper()
	requireGscpBenchmark(b)

	dir := b.TempDir()
	mainPath := filepath.Join(dir, "main.gsc")
	helperPath := filepath.Join(dir, "helpers.gsc")

	writeBenchmarkFile(b, helperPath, "helpers( value ) { }\n")
	baseText := "#include helpers;\n" +
		"main() { helpers(1); }\n"
	altText := "#include helpers;\n" +
		"main() { helpers(2); }\n"
	writeBenchmarkFile(b, mainPath, baseText)

	state := NewState()
	uri := uriForPath(mainPath)
	state.OpenDocument(uri, baseText)

	return &state, uri, baseText, altText
}

func writeBenchmarkFile(b *testing.B, path, content string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("write %s: %v", path, err)
	}
}

func BenchmarkUpdateDocumentWithIncludesNoChange(b *testing.B) {
	state, uri, baseText, _ := setupBenchmarkState(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.UpdateDocument(uri, baseText)
	}
}

func BenchmarkUpdateDocumentWithIncludesSmallEdit(b *testing.B) {
	state, uri, baseText, altText := setupBenchmarkState(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			state.UpdateDocument(uri, altText)
			continue
		}
		state.UpdateDocument(uri, baseText)
	}
}

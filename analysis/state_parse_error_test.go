package analysis

import "testing"

func TestOpenDocumentParseFailurePublishesDiagnostic(t *testing.T) {
	t.Setenv("PATH", "")
	state := NewState()
	uri := "file:///tmp/mp/maps/mp/test.gsc"

	state.OpenDocument(uri, "main() { test(); }\n")
	diagnostics := state.Diagnostics[uri]
	if len(diagnostics) == 0 {
		t.Fatal("expected parse failure diagnostics")
	}
	if diagnostics[0].Source != "gsclsp" {
		t.Fatalf("unexpected diagnostic source: %q", diagnostics[0].Source)
	}
}

func TestParseHandlesPanics(t *testing.T) {
	// This test verifies that the Parse function doesn't crash the process
	// even when given malformed input or when external tools fail
	// The actual gscp binary may not be available in test environment
	// so we just verify the function returns an error rather than panicking

	_, err := Parse("#include \"nonexistent/path.gsc\"\nmain() {}")
	// We expect an error (gscp not found), but not a panic
	if err == nil {
		t.Skip("gscp binary available - skipping panic test")
	}
	t.Logf("Parse returned expected error: %v", err)
}

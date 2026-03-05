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

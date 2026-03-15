package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInlayHintsOpenMethodCallAnchorsAfterParen(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name, quest_desc, quest_check ) { }\n")
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)
	label := "quest_id: "
	hint, ok := findInlayHintByLabel(response.Result, label)
	if !ok {
		t.Fatalf("missing inlay hint for open method call: %v", response.Result)
	}

	line := strings.Split(text, "\n")[1]
	// Hint positioned at cursor location (after '(') with PaddingRight
	// This creates space before the hint so cursor appears after '(' but before hint
	expectedCol := strings.LastIndex(line, "(") + 1
	if expectedCol <= 0 {
		t.Fatalf("test setup missing open paren in line: %q", line)
	}

	if hint.Position.Line != 1 {
		t.Fatalf("inlay hint line = %d, want 1", hint.Position.Line)
	}
	if hint.Position.Character != expectedCol {
		t.Fatalf("inlay hint column = %d, want %d (cursor position after '(')", hint.Position.Character, expectedCol)
	}
	if !hint.PaddingRight {
		t.Fatalf("active param hint must have PaddingRight=true")
	}
}

func TestInlayHintsOpenCallAdvancesParamOnComma(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name, quest_desc, quest_check ) { }\n")
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\"id\", \n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	if !hasInlayLabel(response.Result, "quest_name: ") {
		t.Fatalf("expected next param hint after comma: %v", response.Result)
	}

	line := strings.Split(text, "\n")[1]
	// Hint positioned at cursor location (after ',') with PaddingRight
	// The hint should be at or after the comma position
	commaCol := strings.LastIndex(line, ",")
	if commaCol < 0 {
		t.Fatalf("test setup missing comma in line: %q", line)
	}
	if hint, ok := findInlayHintByLabel(response.Result, "quest_name: "); ok {
		// Hint should be positioned at cursor location, which is after the comma
		// Allow for some variance in exact column due to whitespace handling
		if hint.Position.Character < commaCol {
			t.Fatalf("quest_name hint column = %d, should be at or after comma position %d", hint.Position.Character, commaCol)
		}
		if !hint.PaddingRight {
			t.Fatalf("active param hint must have PaddingRight=true")
		}
	}
	if hasInlayLabel(response.Result, "quest_desc: ") {
		t.Fatalf("unexpected later param hint for open call: %v", response.Result)
	}
}

func TestInlayHintsOpenCallStopsAtActiveParam(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name, quest_desc, quest_check ) { }\n")
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\"id\"\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	if !hasInlayLabel(response.Result, "quest_id: ") {
		t.Fatalf("expected first param hint for open call: %v", response.Result)
	}
	if hasInlayLabel(response.Result, "quest_name: ") {
		t.Fatalf("unexpected next param hint without comma: %v", response.Result)
	}
}

func TestInlayHintsOpenCallIgnoresCommaInString(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name, quest_desc, quest_check ) { }\n")
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\"id,\"\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	if !hasInlayLabel(response.Result, "quest_id: ") {
		t.Fatalf("expected first param hint for open call: %v", response.Result)
	}
	if hasInlayLabel(response.Result, "quest_name: ") {
		t.Fatalf("unexpected next param hint for comma in string: %v", response.Result)
	}
}

func TestInlayHintsOpenCallHavePaddingRight(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name, quest_desc, quest_check ) { }\n")
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	for _, hint := range response.Result {
		if strings.HasSuffix(hint.Label, ": ") && !hint.PaddingRight {
			t.Fatalf("expected open call inlay hint to have PaddingRight=true, got false for hint: %v", hint)
		}
	}
}

func TestInlayHintsOpenCallOnlyActiveParamHasPadding(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name, quest_desc, quest_check ) { }\n")
	// First param already typed, second param is active
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\"id\", \n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	// quest_id hint (already typed) should NOT have padding
	questIdHint, ok := findInlayHintByLabel(response.Result, "quest_id: ")
	if !ok {
		t.Fatalf("missing quest_id hint: %v", response.Result)
	}
	if questIdHint.PaddingRight {
		t.Fatalf("already-typed param quest_id should NOT have PaddingRight, got true")
	}

	// quest_name hint (active) SHOULD have padding
	questNameHint, ok := findInlayHintByLabel(response.Result, "quest_name: ")
	if !ok {
		t.Fatalf("missing quest_name hint: %v", response.Result)
	}
	if !questNameHint.PaddingRight {
		t.Fatalf("active param quest_name should have PaddingRight=true, got false")
	}
}

func TestInlayHintsOpenCallVisualOrderCorrect(t *testing.T) {
	requireGscp(t)
	state := NewState()
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts", "zm")
	mainPath := filepath.Join(scriptsDir, "test.gsc")
	questPath := filepath.Join(scriptsDir, "zm_quests.gsc")

	writeFile(t, questPath, "add_quest( quest_id, quest_name ) { }\n")
	// Open call with first param
	text := "#include scripts\\zm\\zm_quests;\n" +
		"\tself add_quest(\n"
	writeFile(t, mainPath, text)

	uri := uriForPath(mainPath)
	workspaceRoot := "file://" + root
	state.SetWorkspaceFolders([]string{workspaceRoot})
	state.OpenDocument(uri, text)
	response := state.InlayHints(1, uri)

	// Verify visual order: ( -> hint -> space -> cursor
	// This means: hint.Position.Character should be at '(' position
	// With paddingRight, editor renders: ( + hint + space + cursor
	line := strings.Split(text, "\n")[1]
	parenCol := strings.LastIndex(line, "(")
	if parenCol < 0 {
		t.Fatalf("test setup missing paren: %q", line)
	}

	hint, ok := findInlayHintByLabel(response.Result, "quest_id: ")
	if !ok {
		t.Fatalf("missing quest_id hint: %v", response.Result)
	}

	// Verify hint is at cursor position (after paren)
	if hint.Position.Character != parenCol+1 {
		t.Fatalf("hint not at cursor position: got %d, want %d (after '(')", hint.Position.Character, parenCol+1)
	}

	// Verify PaddingRight is set
	if !hint.PaddingRight {
		t.Fatalf("hint must have PaddingRight=true for correct visual order")
	}

	// The visual order should now be:
	// Position parenCol:   ( character
	// Position parenCol+1: cursor (user typing position)
	// Position parenCol+1: hint "quest_id: " (with PaddingRight, appears after cursor in UI)
	//
	// Result: add_quest(|quest_id: ) where | is cursor, but due to PaddingRight,
	// the editor renders: ( + [space] + hint, so cursor appears before hint
}

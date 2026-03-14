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
	expectedCol := strings.LastIndex(line, "(") + 1
	if expectedCol <= 0 {
		t.Fatalf("test setup missing open paren in line: %q", line)
	}

	if hint.Position.Line != 1 {
		t.Fatalf("inlay hint line = %d, want 1", hint.Position.Line)
	}
	if hint.Position.Character != expectedCol {
		t.Fatalf("inlay hint column = %d, want %d", hint.Position.Character, expectedCol)
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
	commaCol := strings.LastIndex(line, ",")
	if commaCol < 0 {
		t.Fatalf("test setup missing comma in line: %q", line)
	}
	commaCol++
	for commaCol < len(line) && (line[commaCol] == ' ' || line[commaCol] == '\t') {
		commaCol++
	}
	if hint, ok := findInlayHintByLabel(response.Result, "quest_name: "); ok {
		if hint.Position.Character != commaCol {
			t.Fatalf("quest_name hint column = %d, want %d", hint.Position.Character, commaCol)
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

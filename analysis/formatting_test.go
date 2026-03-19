package analysis

import (
	"regexp"
	"strings"
	"testing"

	"github.com/maxvanasten/gsclsp/lsp"
)

func TestFormattingReturnsEditsForUnformattedDocument(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){wait 0.05;}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(1, uri, lsp.FormattingOptions{TabSize: 2, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if response.Result[0].NewText == state.Documents[uri] {
		t.Fatal("expected formatting output to differ from original")
	}
}

func TestFormattingUsesRequestedSpaceIndentWidth(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(1){wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(4, uri, lsp.FormattingOptions{TabSize: 8, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if !strings.Contains(response.Result[0].NewText, "\n        if (1) {\n") {
		t.Fatalf("expected 8-space indentation, got: %q", response.Result[0].NewText)
	}
}

func TestFormattingUsesFallbackSpaceIndentWidth(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(1){wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(5, uri, lsp.FormattingOptions{})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if !strings.Contains(response.Result[0].NewText, "\n    if (1) {\n") {
		t.Fatalf("expected fallback 4-space indentation, got: %q", response.Result[0].NewText)
	}
}

func TestFormattingUsesTabsWhenInsertSpacesDisabled(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(1){wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(6, uri, lsp.FormattingOptions{InsertSpaces: false, TabSize: 8})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	if !strings.Contains(response.Result[0].NewText, "\n\tif (1) {\n") {
		t.Fatalf("expected tab indentation, got: %q", response.Result[0].NewText)
	}
}

func TestFormattingPreservesComments(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\n// keep line\nwait 0.05;\n/# keep block #/\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(7, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "// keep line") {
		t.Fatalf("expected line comment to be preserved, got: %q", formatted)
	}
	if !strings.Contains(formatted, "/# keep block #/") {
		t.Fatalf("expected block comment to be preserved, got: %q", formatted)
	}
}

func TestFormattingPreservesSingleBlankLineBetweenTopLevelNodes(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "foo(){wait 0.05;}\n\nbar(){wait 0.05;}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(8, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "}\n\nbar()") {
		t.Fatalf("expected a single blank line between top-level nodes, got: %q", formatted)
	}
}

func TestFormattingCollapsesMultipleBlankLinesBetweenTopLevelNodes(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "foo(){wait 0.05;}\n\n\n\nbar(){wait 0.05;}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(9, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if strings.Contains(formatted, "\n\n\n") {
		t.Fatalf("expected multiple blank lines to be collapsed, got: %q", formatted)
	}
	if !strings.Contains(formatted, "}\n\nbar()") {
		t.Fatalf("expected one blank line to remain between top-level nodes, got: %q", formatted)
	}
}

func TestFormattingPreservesSingleBlankLineWithinFunctionBody(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nwait 0.05;\n\nwait 0.05;\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(10, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "wait 0.05;\n\n    wait 0.05;") {
		t.Fatalf("expected one blank line between statements in scope, got: %q", formatted)
	}
}

func TestFormattingCollapsesMultipleBlankLinesWithinFunctionBody(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nwait 0.05;\n\n\n\nwait 0.05;\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(11, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if strings.Contains(formatted, "\n\n\n") {
		t.Fatalf("expected multiple blank lines in scope to be collapsed, got: %q", formatted)
	}
	if !strings.Contains(formatted, "wait 0.05;\n\n    wait 0.05;") {
		t.Fatalf("expected one blank line to remain between statements in scope, got: %q", formatted)
	}
}

func TestFormattingCollapsesMultipleBlankLinesAroundNestedBlockInFunctionBody(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nwait 0.05;\n\n\nif(1){\nwait 0.05;\n}\n\n\n\nwait 0.05;\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(13, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if strings.Contains(formatted, "\n\n\n") {
		t.Fatalf("expected multiple blank lines in nested scope context to be collapsed, got: %q", formatted)
	}
	if !strings.Contains(formatted, "wait 0.05;\n\n    if (1)") {
		t.Fatalf("expected one blank line before nested block, got: %q", formatted)
	}
	if !strings.Contains(formatted, "}\n\n    wait 0.05;") {
		t.Fatalf("expected one blank line after nested block, got: %q", formatted)
	}
}

func TestFormattingPreservesSingleBlankLineWithinSwitchCaseBody(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nswitch(a){\ncase 1:\nwait 0.05;\n\n\n\nwait 0.05;\nbreak;\n}\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(14, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	multipleBlankLinesPattern := regexp.MustCompile(`\n[ \t]*\n[ \t]*\n`)
	if multipleBlankLinesPattern.MatchString(formatted) {
		t.Fatalf("expected multiple blank lines in switch case body to be collapsed, got: %q", formatted)
	}
	singleBlankLineBetweenWaitsPattern := regexp.MustCompile(`wait 0\.05;\n[ \t]*\n[ \t]*wait 0\.05;`)
	if !singleBlankLineBetweenWaitsPattern.MatchString(formatted) {
		t.Fatalf("expected one blank line to remain inside switch case body, got: %q", formatted)
	}
}

func TestFormattingPreservesThreadFunctionPointerSyntax(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	input := "main(){self thread [[ self.gobblegum.usefunc ]]();}"
	state.Documents[uri] = input

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(20, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "thread [[ self.gobblegum.usefunc ]]()") {
		t.Fatalf("expected thread function pointer syntax to be preserved, got: %q", formatted)
	}
}

func TestFormattingPreservesNonThreadFunctionPointerSyntax(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	input := "main(){self [[ self.gobblegum.usefunc ]]();}"
	state.Documents[uri] = input

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(21, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "[[ self.gobblegum.usefunc ]]()") {
		t.Fatalf("expected function pointer syntax to be preserved, got: %q", formatted)
	}
}

func TestFormattingPreservesMethodQualifierOnFunctionCalls(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nself.somefunc();\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(12, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if strings.Contains(formatted, "\n    somefunc();") {
		t.Fatalf("expected method qualifier to be preserved, got: %q", formatted)
	}
	methodCallPattern := regexp.MustCompile(`self[ .]+somefunc\(\);`)
	if !methodCallPattern.MatchString(formatted) {
		t.Fatalf("expected formatted output to preserve qualified method call, got: %q", formatted)
	}
}

func TestFormattingDoesNotDoubleTerminateFunctionCalls(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\narray(\"specialty_quickrevive\", \"specialty_deadshot\", \"specialty_fastreload\", \"specialty_armorvest\", \"specialty_longersprint\", \"specialty_rof\", \"specialty_grenadepulldeath\");\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(15, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	doubleTerminatorPattern := regexp.MustCompile(`array\([^\n]*\);;`)
	if doubleTerminatorPattern.MatchString(formatted) {
		t.Fatalf("expected no double terminator on function call, got: %q", formatted)
	}
	singleTerminatorPattern := regexp.MustCompile(`array\([^\n]*\);`)
	if !singleTerminatorPattern.MatchString(formatted) {
		t.Fatalf("expected function call to end with one terminator, got: %q", formatted)
	}
}

func TestFormattingDoesNotDoubleTerminateReturnCallInSwitchCase(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nswitch(level.mapname){\ncase \"zm_nuked\":\nreturn array(\"m1911_zm\", \"m1911_upgraded_zm\");;\n}\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(18, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if strings.Contains(formatted, ";;") {
		t.Fatalf("expected no double terminator in switch-case return call, got: %q", formatted)
	}
	if !strings.Contains(formatted, "return array(\"m1911_zm\", \"m1911_upgraded_zm\");") {
		t.Fatalf("expected switch-case return call to keep single terminator, got: %q", formatted)
	}
}

func TestFormattingPreservesUnaryMinusExpressionInsideFunctionCallArgs(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){\nself.hud_perks[i]=ml_create_text(1.25,-200,-130-i*10,\"\");\n}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(19, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if strings.Contains(formatted, "-130, , i * 10") {
		t.Fatalf("expected unary-minus binary expression to stay intact in function args, got: %q", formatted)
	}
	if !strings.Contains(formatted, "ml_create_text(1.25, -200, -130 - i * 10, \"\");") {
		t.Fatalf("expected formatted function args to include '-130 - i * 10' expression, got: %q", formatted)
	}
}

func TestFormattingInlinesElseAfterIfBlock(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(x){wait 0.05;}else{wait 0.05;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(16, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "if (x) {") {
		t.Fatalf("expected if opening brace to be inline, got: %q", formatted)
	}
	if strings.Contains(formatted, "}\n\n    else") {
		t.Fatalf("expected no blank line before else, got: %q", formatted)
	}
	if !strings.Contains(formatted, "} else {") {
		t.Fatalf("expected if/else chain with inline braces, got: %q", formatted)
	}
}

func TestFormattingInlinesElseIfOnSingleLine(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){if(x){wait 0.05;}else if(y){wait 0.1;}else{wait 0.2;}}"

	ensureParserAvailable(t, state.Documents[uri])

	response := state.Formatting(17, uri, lsp.FormattingOptions{TabSize: 4, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if !strings.Contains(formatted, "} else if (y)") {
		t.Fatalf("expected else if to be on one line, got: %q", formatted)
	}
	if !strings.Contains(formatted, "} else if (y) {") {
		t.Fatalf("expected else-if opening brace to be inline, got: %q", formatted)
	}
	if !strings.Contains(formatted, "} else {") {
		t.Fatalf("expected final else opening brace to be inline, got: %q", formatted)
	}
	if strings.Contains(formatted, "} else\n    if (y)") {
		t.Fatalf("expected else and if to stay inline, got: %q", formatted)
	}
}

func TestFormattingReturnsNoEditsOnParseFailure(t *testing.T) {
	t.Setenv("PATH", "")
	state := NewState()
	uri := "file:///tmp/test.gsc"
	state.Documents[uri] = "main(){wait 0.05;}"

	response := state.Formatting(2, uri, lsp.FormattingOptions{})
	if len(response.Result) != 0 {
		t.Fatalf("expected no edits on parse failure, got %d", len(response.Result))
	}
}

func TestFormattingReturnsNoEditsForMissingDocument(t *testing.T) {
	state := NewState()
	response := state.Formatting(3, "file:///tmp/missing.gsc", lsp.FormattingOptions{})
	if len(response.Result) != 0 {
		t.Fatalf("expected no edits for missing document, got %d", len(response.Result))
	}
}

func TestFormattingPreservesMultilineArrays(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	input := `x = [
  1
  2
];`
	expected := `x = [
  1,
  2
];`

	state.Documents[uri] = input
	ensureParserAvailable(t, input)

	response := state.Formatting(4, uri, lsp.FormattingOptions{TabSize: 2, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if formatted != expected {
		t.Fatalf("expected multiline array preservation:\n%s\ngot:\n%s", expected, formatted)
	}
}

func TestFormattingPreservesMultilineFunctionCalls(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	input := `my_func(
  arg1
  arg2
);`
	expected := `my_func(
  arg1,
  arg2
);`

	state.Documents[uri] = input
	ensureParserAvailable(t, input)

	response := state.Formatting(5, uri, lsp.FormattingOptions{TabSize: 2, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if formatted != expected {
		t.Fatalf("expected multiline function call preservation:\n%s\ngot:\n%s", expected, formatted)
	}
}

func TestFormattingKeepsSingleLineArrays(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	input := `x = [1, 2, 3];`

	state.Documents[uri] = input
	ensureParserAvailable(t, input)

	response := state.Formatting(6, uri, lsp.FormattingOptions{TabSize: 2, InsertSpaces: true})
	// No edits should be returned for already-formatted single-line array
	if len(response.Result) != 0 {
		t.Fatalf("expected no edits for single-line array, got %d", len(response.Result))
	}
}

func TestFormattingPreservesMultilineVectorLiterals(t *testing.T) {
	state := NewState()
	uri := "file:///tmp/test.gsc"
	// Input has inconsistent indentation that should be fixed
	input := `pos = (
0,
1,
2
);`
	expected := `pos = (
  0,
  1,
  2
);`

	state.Documents[uri] = input
	ensureParserAvailable(t, input)

	response := state.Formatting(7, uri, lsp.FormattingOptions{TabSize: 2, InsertSpaces: true})
	if len(response.Result) != 1 {
		t.Fatalf("expected one formatting edit, got %d", len(response.Result))
	}
	formatted := response.Result[0].NewText
	if formatted != expected {
		t.Fatalf("expected multiline vector literal preservation:\n%s\ngot:\n%s", expected, formatted)
	}
}

func ensureParserAvailable(t *testing.T, input string) {
	t.Helper()
	if _, err := Parse(input); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			t.Skip("gscp not available on PATH")
		}
		t.Fatalf("parse precheck failed: %v", err)
	}
}

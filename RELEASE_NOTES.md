# Release Notes

## 0.9.0

- **Enhancement**: Multiline array and function call formatting preservation (#28)
  - Parser now supports multiline arrays: `array(\n  "a"\n  "b"\n)`
  - Parser now supports multiline function arguments: `func(\n  arg1\n  arg2\n)`
  - Formatter preserves multiline style with proper indentation
  - Single-line arrays and calls remain compact
  - Vector literals also support multiline formatting

## 0.8.8

- **Bug Fix**: Prevent LSP crashes and hangs when editing strings and incomplete code (#27)
  - Add panic recovery throughout message handling and diagnostic generation
  - Add 5-second timeout to gscp parser to prevent hanging on malformed input
  - Clear stale AST/state before parsing to prevent diagnostic corruption
  - Fix recursion bug in `GenerateFunctionSignatures` that could cause hangs
  - Add panic recovery in `SemanticTokens` and signature generation
  - Truncate long error messages to prevent oversized diagnostics

## 0.8.7

- **Bug Fix**: Fixed formatting of function pointer syntax `[[ expression ]]()` in thread calls (#25)
- **Bug Fix**: Fixed inlay hints not updating when functions in included files are modified (#26)
- **Enhancement**: Improved cache invalidation with include file modification time tracking

## 0.8.6

- **Bug Fix**: Fixed workspace includes on Windows by handling drive letter paths correctly in `uriToPath()` and `pathToURI()` functions (#23)

## 0.8.5

- **Enhancement**: Expanded builtin function signatures from 56 to 1792 (1,736 new builtin functions)

## 0.8.4

- **Bug Fix**: Add `paddingRight` to active parameter inlay hints for open calls, improving visual spacing between hints and subsequent code
- **Testing**: Add comprehensive test coverage for open call inlay hints (7 new tests covering positioning, comma advancement, string handling, and padding behavior)

## 0.8.3

- **Bug Fix**: Improve open-call inlay hints to advance with commas, ignore commas inside strings, and anchor active hints after the current comma
- **Enhancement**: Resolve includes and definitions using workspace folders with auto-detection during initialization

## 0.8.2

- **Bug Fix**: Fix race condition with concurrent map access - add RWMutex and cache mutex to protect concurrent map access, fixing 'concurrent map read and map write' panic

## 0.8.1

- **Performance**: Implement incremental document sync (LSP textDocumentSync mode 2) to receive text changes instead of full document content
- **Performance**: Add lazy parsing - AST is now parsed on-demand when LSP features are requested instead of on every keystroke
- **Performance**: Debounced diagnostics publishing (150ms delay) to reduce CPU usage during active typing

## 0.8.0

- Added 39 new engine builtin functions

## 0.7.9

- Updates formatter conditional block layout to keep opening braces inline for `if`, `else if`, and `else` chains (for example `} else {`).
- Fixes formatting output that could preserve duplicate terminators in return expressions (for example `return array(...);;` inside switch/case blocks).

## 0.7.8

- Keeps formatter `if/else` chains inline by emitting `} else` and `} else if (...)` on one line instead of splitting `else` onto a separate block.
- Updates local build workflow to write artifacts under `dist/` so routine builds no longer leave an untracked `gsclsp` binary in the repository root.

## 0.7.7

- Fixes formatting output that could append a duplicate statement terminator on already-terminated function calls (for example `array(...);;`).
- Switches active release versioning from `0.0.7.x` to `0.7.x`.

## 0.0.7.6

- Preserves intentional single blank lines inside formatted function bodies while still collapsing larger blank-line runs.
- Extends original-spacing formatting logic to switch/case scopes so blank lines are retained consistently within case bodies.

## 0.0.7.5

- Reduces receiver `self` context inlay hint clutter by emitting one function-level hint instead of repeating hints after each `self` use.
- Repositions the receiver context hint to render after function declaration parentheses for clearer signature-level context.

## 0.0.7.4

- Extends `self` context inlay hints to cover `self.property` access patterns so receiver context also appears on property method calls.
- Refines property hint labels to append directly to each inferred receiver (for example ` -> level.weapon, player.weapon`) to avoid ambiguous interpretation.

## 0.0.7.3

- Updates `self` context inlay hints to show combined receiver candidates for ambiguous call paths instead of hiding the hint.
- Caps combined receiver output at three entries and appends `...` when more candidates exist (for example ` -> a, b, c, ...`).

## 0.0.7.2

- Preserves a single intentional blank line between top-level statements during formatting while still collapsing larger blank-line runs to one.
- Adds `self` context inline hints inferred from unambiguous threaded call receivers, rendered after `self` as ` -> receiver`.
- Fixes `self` receiver propagation across nested `self` calls so downstream inlay hints keep the concrete receiver context (for example `player`) instead of falling back to `self`.

## 0.0.7.1

- Fixes include-origin inlay hint placement for method calls so origin labels render before the function name instead of before the method receiver object.
- Rewrites the open-call include inlay regression test to use temporary fixtures, removing dependence on local `test/two.gsc` files and making full test runs reliable across environments.

## 0.0.7

- Adds stdlib go-to-definition that opens generated read-only `.gsc` files containing full stdlib function declarations and bodies.
- Extends stdlib generation to embed declaration text alongside signatures, including qualified stdlib definition resolution.
- Cleans up stdlib definition temp directories reliably with process-exit cleanup and PID-aware stale directory pruning on startup.
- Improves large-file responsiveness by reducing expensive payload logging, deferring include resolution, and optimizing inlay hint resolution paths.

## 0.0.6.9

- Adds include-origin inlay hints for unqualified included function calls, shown as `include\\path::` at the call site.
- Keeps origin hints hidden for local declarations, builtins, and already-qualified calls to avoid redundant noise.
- Restricts code actions to `quickfix` only so VS Code surfaces them as regular code actions instead of source actions.

## 0.0.6.8

- Fixes VS Code code action discovery by improving code action kind filtering behavior for client `only` requests.
- Fixes path completion replacement to correctly replace typed include/path prefixes instead of appending duplicate segments.
- Expands stdlib generation with optional map roots that scan each map's `maps/mp` runtime subtree.
- Normalizes map script signatures to runtime include keys under `maps/mp/...` and reports duplicate stdlib keys with their source maps.

## 0.0.6.7

- Adds LSP code action support with `textDocument/codeAction` and `workspace/executeCommand`.
- Introduces `gsclsp.bundleMod` to bundle scripts into a nested mod folder named after the current directory.
- Rebuilds bundle output from scratch on each run to remove stale files from previous bundles.
- Recursively copies `.gsc` files while preserving relative paths, skipping hidden directories, and keeping originals untouched.
- Advertises comment semantic tokens in the legend and adds regression coverage for comment highlighting and bundling behavior.

## 0.0.6.5

- Adds LSP `textDocument/completion` capability advertisement and request routing.
- Implements contextual completions for functions, keywords, include paths, and qualified path/function calls.
- Adds snippet-style function insertion text with parameter placeholders for faster call authoring.
- Expands regression coverage for completion routing, capability advertisement, and completion context behavior.

## 0.0.6.4

- Fixes LSP method routing for `textDocument/definition` while keeping legacy compatibility.
- Implements real go-to-definition for local declarations and local include trees.
- Replaces fatal parser exits with recoverable errors and diagnostic fallback behavior.
- Improves include parsing performance with a file metadata cache and unchanged-document short circuit.
- Adds portable tracked build scripts in `scripts/` with env-driven stdlib generation inputs.
- Expands regression coverage for definition behavior, include cache invalidation, and update benchmarks.

# Release Notes

## 0.0.6.10

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

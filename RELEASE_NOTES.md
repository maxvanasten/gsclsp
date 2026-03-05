# Release Notes

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

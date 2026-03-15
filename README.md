# gsclsp

`gsclsp` is a Language Server Protocol (LSP) server for `.gsc` scripts used in older Call of Duty titles.

It runs over standard LSP stdio (`stdin`/`stdout`) and uses [`gscp`](https://github.com/maxvanasten/gscp) for parsing and diagnostics.

VS Code extension: [GSCLSP for GSC](https://marketplace.visualstudio.com/items?itemName=maxvanasten.gsclsp-vscode)

## Features

`gsclsp` currently provides:

- Semantic tokens (syntax-aware highlighting)
- Hover signatures for function calls
- Inlay hints for function call arguments
- Go to definition for local, included, and stdlib functions
- Diagnostics from the `gscp` parser
- Document formatting (`textDocument/formatting`)
- Code actions for bundling scripts into mod structure (`textDocument/codeAction` + `workspace/executeCommand`)

## Supported LSP capabilities

During `initialize`, the server advertises:

- `textDocumentSync`: full document sync (`1`)
- `hoverProvider`: `true`
- `definitionProvider`: `true`
- `documentFormattingProvider`: `true`
- `codeActionProvider`: `true`
- `executeCommandProvider`: `gsclsp.bundleMod`
- `semanticTokensProvider`: full document support (`full: true`, `range: false`)
- `inlayHintProvider`: `true`

Semantic token legend:

- `variable`
- `keyword`
- `string`
- `number`
- `function`
- `property`
- `comment`

Bundle code action behavior:

- Creates a nested mod folder named after the current directory
- Replaces the existing nested mod folder on each run (no stale leftovers)
- Writes `<modName>/mod.json` with default metadata
- Recursively copies `.gsc` files into `<modName>/scripts` while preserving relative paths
- Skips hidden directories (for example `.git`)
- Keeps original source `.gsc` files in place

## How it works

At a high level:

1. `gsclsp` receives LSP messages over stdio.
2. On document open/change, it invokes `gscp` to parse text.
3. It stores AST/tokens/signatures per document in memory.
4. It augments signatures with embedded builtins and stdlib signature bundles.
5. It serves hover/definition/inlay/semantic-token responses from this analysis state.

Include handling details:

- Supports local `#include` resolution relative to the current file URI
- Supports recursive include traversal for definition lookup
- Uses include file caching (mtime + file size) to avoid reparsing unchanged includes
- Supports qualified calls like `maps\\mp\\zombies\\_zm_utility::init_utility`
- Uses URI path heuristics (`/mp/` or `/zm/`) to prefer matching stdlib groups

## Requirements

- Go `1.25+` (module currently targets `go 1.25.7`)
- [`gscp`](https://github.com/maxvanasten/gscp) installed and available on `PATH`

`gscp` is required at runtime for parsing and diagnostics.

## Installation

### Option 1: VS Code (recommended for VS Code users)

Install from the marketplace:

- [GSCLSP for GSC](https://marketplace.visualstudio.com/items?itemName=maxvanasten.gsclsp-vscode)

The extension manages compatible `gsclsp` and `gscp` releases for you.

### Option 2: Build from source

```bash
git clone https://github.com/maxvanasten/gsclsp
cd gsclsp
go build -o dist/gsclsp ./
```

This produces a `gsclsp` binary at `dist/gsclsp`.

Optional install:

```bash
sudo mv ./dist/gsclsp /usr/local/bin/gsclsp
```

Helper scripts:

```bash
./scripts/dev-check.sh
./scripts/build-local.sh
./scripts/build-releases.sh
```

Script environment variables are documented in `scripts/README.md`.

## Neovim setup

Example Neovim LSP configuration:

```lua
vim.filetype.add({
  extension = {
    gsc = "gsc",
  },
})

vim.lsp.config["gsclsp"] = {
  cmd = { "gsclsp" },
  filetypes = { "gsc" },
  single_file_support = true,
}

vim.lsp.enable({ "gsclsp" })
```

If `gsclsp` is not on your `PATH`, replace `cmd` with an absolute binary path.

## Stdlib signature generation

The repository embeds signature bundles in `analysis/stdlib_signatures.json`, declaration bundles in `analysis/stdlib_declarations.json`, and builtins in `analysis/builtins_signatures.json`.

To regenerate stdlib signatures from local script roots:

```bash
go run ./cmd/stdlibgen \
  --mp-root "/path/to/t6-source/mp/core" \
  --zm-root "/path/to/t6-source/zm/core" \
  --mp-maps-root "/path/to/t6-source/mp/maps" \
  --zm-maps-root "/path/to/t6-source/zm/maps" \
  --out "analysis/stdlib_signatures.json" \
  --out-declarations "analysis/stdlib_declarations.json"
```

Notes:

- Both `--mp-root` and `--zm-root` are required.
- `--mp-maps-root` and `--zm-maps-root` are optional and include map-specific scripts.
- Map roots scan each map directory's `maps/mp` subtree only.
- Map-specific scripts are normalized to runtime include keys under `maps/mp/...`.
- The generator walks `.gsc` files, parses with `gscp`, and writes a JSON bundle keyed by normalized include paths.

## Development

Run tests:

```bash
go test ./...
```

Important test notes:

- Many analysis tests require `gscp` to be installed.
- Include-based inlay tests create temporary fixture files at runtime and do not require repo-local test fixture files.

## Release checklist

Before building a new release:

1. Update the version in `lsp/initialize.go` and `README.md`.
2. Add release notes to `RELEASE_NOTES.md`.
3. Run quality checks:

   ```bash
   ./scripts/dev-check.sh
   ```

4. Build release artifacts:

   ```bash
   ./scripts/build-releases.sh
   ```

5. Create a draft GitHub release and attach files from `dist/`.

## Project structure

- `main.go`: LSP message loop and request routing
- `analysis/`: parser integration, signatures, diagnostics, hover/definition/inlay/semantic token logic
- `lsp/`: LSP request/response structs and capability definitions
- `rpc/`: LSP framing (`Content-Length`) encode/decode helpers
- `cmd/stdlibgen/`: CLI tool to generate stdlib signature bundles

## Version

Current server version reported in `initialize` response: `0.8.5`.

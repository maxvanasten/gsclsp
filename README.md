# GSCLSP

[![Version](https://img.shields.io/github/v/release/maxvanasten/gsclsp)](https://github.com/maxvanasten/gsclsp/releases)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen)]()

Language intelligence for Call of Duty GSC scripting.

GSCLSP is a [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) implementation for `.gsc` files used in older Call of Duty titles. It provides IDE features like code completion, diagnostics, and go-to-definition for GSC scripts.

## Features

- **Intelligent Code Completion** — Function signatures and parameter hints as you type
- **Go to Definition** — Jump to function definitions across files and includes
- **Real-time Diagnostics** — Syntax errors shown inline as you code
- **Auto-formatting** — Consistent code formatting with configurable indentation
- **Semantic Highlighting** — Syntax-aware token coloring for better readability
- **Include Support** — Full support for `#include` statements with recursive resolution

## Installation

### VS Code

The easiest way to get started. The extension auto-downloads and manages `gsclsp` and its parser dependency.

1. Install from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=maxvanasten.gsclsp-vscode)
2. Open any `.gsc` file — the extension handles the rest

**Configuration options:**
- `gsclsp.path` — Use a custom `gsclsp` binary instead of auto-download
- `gsclsp.updates.enabled` — Enable/disable automatic updates (default: `true`)
- `gsclsp.updates.check` — Update check frequency: `always`, `daily`, `weekly`, `never`

### Neovim

Using [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig):

```lua
require('lspconfig').gsclsp.setup{
  cmd = { "gsclsp" },
  filetypes = { "gsc" },
}
```

Or with the native LSP client (Neovim 0.11+):

```lua
vim.lsp.config['gsclsp'] = {
  cmd = { "gsclsp" },
  filetypes = { "gsc" },
  single_file_support = true,
}
vim.lsp.enable('gsclsp')
```

### Manual Installation

Download a pre-built binary from [GitHub Releases](https://github.com/maxvanasten/gsclsp/releases):

```bash
# Linux x64
wget https://github.com/maxvanasten/gsclsp/releases/latest/download/gsclsp-linux-x64 -O gsclsp
chmod +x gsclsp
sudo mv gsclsp /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/maxvanasten/gsclsp/releases/latest/download/gsclsp-darwin-x64 -o gsclsp
chmod +x gsclsp
sudo mv gsclsp /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/maxvanasten/gsclsp/releases/latest/download/gsclsp-darwin-arm64 -o gsclsp
chmod +x gsclsp
sudo mv gsclsp /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/maxvanasten/gsclsp/releases/latest/download/gsclsp-win32-x64.exe -OutFile gsclsp.exe
# Move to a directory in your PATH
```

**Building an extension for another editor?** gsclsp speaks standard LSP over stdio — check the [VS Code extension source](https://github.com/maxvanasten/gsclsp-vscode) as a reference implementation.

## How it Works

GSCLSP runs as a language server over standard input/output, communicating via the Language Server Protocol. It uses [`gscp`](https://github.com/maxvanasten/gscp) for parsing GSC files into ASTs, then provides IDE features like hover information, diagnostics, and go-to-definition based on that analysis.

## Development

### Prerequisites

- Go 1.25+
- [`gscp`](https://github.com/maxvanasten/gscp) installed on PATH (required at runtime)

### Building

```bash
# Local build
./scripts/build-local.sh

# Cross-platform releases
./scripts/build-releases.sh

# With stdlib regeneration from CoD sources
./scripts/build-local.sh  # or build-releases.sh
```

### Testing

```bash
go test ./...
```

Note: Many tests require `gscp` to be installed on PATH.

## Project Structure

```
├── analysis/       # Core language analysis (parsing, signatures, diagnostics)
├── cmd/stdlibgen/  # Tool to generate stdlib signatures from CoD sources
├── lsp/            # LSP protocol types and responses
├── rpc/            # LSP message framing (Content-Length)
└── scripts/        # Build and development scripts
```

## Version

Current version: **0.8.7**

See [RELEASE_NOTES.md](RELEASE_NOTES.md) for detailed changelog.

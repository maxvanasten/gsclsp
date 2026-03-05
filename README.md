# gsclsp

`gsclsp` is a language server implementation for the .gsc language used in older Call of Duty titles.

[VSCode extension](https://marketplace.visualstudio.com/items?itemName=maxvanasten.gsclsp-vscode)

## Features

### Implemented

- Semantic tokens (provides syntax highlighting)
- Completion
- Inline hints (arguments in function calls)
- Hover definitions for function calls (will show the function signature)
- Diagnostics

## Installation

#### Dependencies
`gsclsp` requires the latest version of [gscp](https://github.com/maxvanasten/gscp) installed on your path, if you use the vscode extension, the appropriate releases of gsclsp and gscp are automatically installed for you.

### Neovim

#### Build

```bash
# Clone gsclsp
git clone https://github.com/maxvanasten/gsclsp
# Build gsclsp from source
go build
# (OPTIONAL) Move gsclsp to /usr/bin so its accessible anywhere
sudo mv ./gsclsp /usr/bin/gsclsp

```

#### Portable build scripts

The repository now includes tracked, portable scripts under `scripts/`.

```bash
# Build with stdlib regeneration
GSCLSP_MP_ROOT="/path/to/t6-source/mp/core" \
GSCLSP_ZM_ROOT="/path/to/t6-source/zm/core" \
./scripts/build.sh

# Build without regenerating stdlib signatures
GSCLSP_SKIP_STDLIBGEN=1 ./scripts/build.sh

# Optional install target path
GSCLSP_INSTALL_PATH="/usr/local/bin/gsclsp" \
GSCLSP_SKIP_STDLIBGEN=1 \
./scripts/build.sh
```

```bash
# Cross-platform release artifacts in ./dist
GSCLSP_MP_ROOT="/path/to/t6-source/mp/core" \
GSCLSP_ZM_ROOT="/path/to/t6-source/zm/core" \
./scripts/build-releases.sh
```

#### stdlib signature generation

You can also run the generator directly:

```bash
go run ./cmd/stdlibgen \
  --mp-root "/path/to/t6-source/mp/core" \
  --zm-root "/path/to/t6-source/zm/core" \
  --out "analysis/stdlib_signatures.json"
```

#### .config/nvim

```lua
vim.filetype.add({
  extension = {
    gsc = 'gsc',
  },
})

vim.lsp.config['gsclsp'] = {
  cmd = { 'gsclsp' }, -- Or relative filepath if you have not moved ./gsclsp to /usr/bin
  filetypes = { 'gsc' },
  single_file_support = true,
}

vim.lsp.enable({'gsclsp'})
```

### VSCode

You can download the extension by searching for "GSCLSP for GSC" by maxvanasten or by installing it from the marketplace: [GSCLSP for GSC](https://marketplace.visualstudio.com/items?itemName=maxvanasten.gsclsp-vscode)

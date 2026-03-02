# gsclsp

`gsclsp` is a language server implementation for the .gsc language used in older Call of Duty titles.

## Features

### Implemented

- Semantic tokens (provides syntax highlighting)

### Planned

- Completion
- Inline hints (arguments in function calls)
- Hover definitions for function calls (will show the function signature)

## Installation

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

WIP.

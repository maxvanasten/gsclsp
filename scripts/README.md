# Scripts

## dev-check

Runs local quality checks:

```bash
./scripts/dev-check.sh
```

## build-local

Builds and installs `gsclsp` locally:

```bash
./scripts/build-local.sh
```

Environment variables:

- `MP_ROOT` (default `/home/max/Code/t6-source/mp/core`)
- `ZM_ROOT` (default `/home/max/Code/t6-source/zm/core`)
- `MP_MAPS_ROOT` (default `/home/max/Code/t6-source/mp/maps`, falls back to `/home/max/Code/t6-source/mp/Maps`)
- `ZM_MAPS_ROOT` (default `/home/max/Code/t6-source/zm/maps`, falls back to `/home/max/Code/t6-source/zm/Maps`)
- `INSTALL_PATH` (default `/usr/local/bin/gsclsp`)
- `SKIP_STDLIB_GEN` (`1` or `true` to skip stdlib generation)

## build-releases

Builds cross-platform release binaries into `dist/`:

```bash
./scripts/build-releases.sh
```

Environment variables are the same as `build-local.sh`.

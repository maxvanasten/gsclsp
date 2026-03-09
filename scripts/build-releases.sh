#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(pwd)"
OUT_DIR="${ROOT_DIR}/dist"

MP_ROOT="${MP_ROOT:-/home/max/Code/t6-source/mp/core}"
ZM_ROOT="${ZM_ROOT:-/home/max/Code/t6-source/zm/core}"

MP_MAPS_ROOT="${MP_MAPS_ROOT:-/home/max/Code/t6-source/mp/maps}"
if [ ! -d "${MP_MAPS_ROOT}" ]; then
  MP_MAPS_ROOT="/home/max/Code/t6-source/mp/Maps"
fi

ZM_MAPS_ROOT="${ZM_MAPS_ROOT:-/home/max/Code/t6-source/zm/maps}"
if [ ! -d "${ZM_MAPS_ROOT}" ]; then
  ZM_MAPS_ROOT="/home/max/Code/t6-source/zm/Maps"
fi

mkdir -p "${OUT_DIR}"

if [ "${SKIP_STDLIB_GEN:-}" = "1" ] || [ "${SKIP_STDLIB_GEN:-}" = "true" ]; then
  echo "[build] skipping stdlib generation"
else
  echo "[build] generating stdlib signatures"
  go run ./cmd/stdlibgen \
    --mp-root "${MP_ROOT}" \
    --zm-root "${ZM_ROOT}" \
    --mp-maps-root "${MP_MAPS_ROOT}" \
    --zm-maps-root "${ZM_MAPS_ROOT}" \
    --out "${ROOT_DIR}/analysis/stdlib_signatures.json" \
    --out-declarations "${ROOT_DIR}/analysis/stdlib_declarations.json"
fi

build_target() {
  local goos="$1"
  local goarch="$2"
  local name="$3"
  local arch_label="$4"
  local ext="$5"

  local output="${OUT_DIR}/gsclsp-${name}-${arch_label}${ext}"
  echo "[build] ${output}"
  GOOS="${goos}" GOARCH="${goarch}" go build -o "${output}" ./
}

build_target linux amd64 linux x64 ""
build_target linux arm64 linux arm64 ""
build_target darwin amd64 darwin x64 ""
build_target darwin arm64 darwin arm64 ""
build_target windows amd64 win32 x64 ".exe"

echo "[build] done"

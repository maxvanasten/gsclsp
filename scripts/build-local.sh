#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(pwd)"

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

INSTALL_PATH="${INSTALL_PATH:-/usr/local/bin/gsclsp}"

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

echo "[build] building gsclsp"
go build ./

echo "[build] installing to ${INSTALL_PATH}"
sudo install -m 0755 "${ROOT_DIR}/gsclsp" "${INSTALL_PATH}"

echo "[build] done"

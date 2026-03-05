#!/usr/bin/env bash
set -euo pipefail

start_ts=$(date +%s)

log() {
  echo "[build] $*"
}

finish() {
  status=$?
  end_ts=$(date +%s)
  elapsed=$((end_ts - start_ts))
  if [ "$status" -eq 0 ]; then
    log "Completed in ${elapsed}s"
  else
    log "Failed after ${elapsed}s"
  fi
}

trap finish EXIT

ROOT_DIR="$(pwd)"
STDLIB_OUT="${ROOT_DIR}/analysis/stdlib_signatures.json"

if [ "${GSCLSP_SKIP_STDLIBGEN:-0}" != "1" ]; then
  if [ -z "${GSCLSP_MP_ROOT:-}" ] || [ -z "${GSCLSP_ZM_ROOT:-}" ]; then
    log "Set GSCLSP_MP_ROOT and GSCLSP_ZM_ROOT or use GSCLSP_SKIP_STDLIBGEN=1"
    exit 1
  fi

  step_start=$(date +%s)
  log "Generating stdlib signatures"
  go run ./cmd/stdlibgen --mp-root "${GSCLSP_MP_ROOT}" --zm-root "${GSCLSP_ZM_ROOT}" --out "${STDLIB_OUT}"
  step_end=$(date +%s)
  log "Stdlib generation done in $((step_end - step_start))s"
fi

step_start=$(date +%s)
log "Building gsclsp"
go build
step_end=$(date +%s)
log "Build done in $((step_end - step_start))s"

if [ -n "${GSCLSP_INSTALL_PATH:-}" ]; then
  step_start=$(date +%s)
  log "Installing gsclsp to ${GSCLSP_INSTALL_PATH}"
  install -m 0755 ./gsclsp "${GSCLSP_INSTALL_PATH}"
  step_end=$(date +%s)
  log "Install done in $((step_end - step_start))s"
fi

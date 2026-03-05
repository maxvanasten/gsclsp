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
OUT_DIR="${ROOT_DIR}/dist"

mkdir -p "${OUT_DIR}"

if [ "${GSCLSP_SKIP_STDLIBGEN:-0}" != "1" ]; then
  if [ -z "${GSCLSP_MP_ROOT:-}" ] || [ -z "${GSCLSP_ZM_ROOT:-}" ]; then
    log "Set GSCLSP_MP_ROOT and GSCLSP_ZM_ROOT or use GSCLSP_SKIP_STDLIBGEN=1"
    exit 1
  fi

  step_start=$(date +%s)
  log "Generating stdlib signatures"
  go run ./cmd/stdlibgen --mp-root "${GSCLSP_MP_ROOT}" --zm-root "${GSCLSP_ZM_ROOT}" --out "${ROOT_DIR}/analysis/stdlib_signatures.json"
  step_end=$(date +%s)
  log "Stdlib generation done in $((step_end - step_start))s"
fi

build_target() {
  local goos="$1"
  local goarch="$2"
  local name="$3"
  local arch_label="$4"
  local ext="$5"

  local output="${OUT_DIR}/gsclsp-${name}-${arch_label}${ext}"
  local target_start
  local target_end

  log "Building ${output}"
  target_start=$(date +%s)
  GOOS="${goos}" GOARCH="${goarch}" go build -o "${output}" ./
  target_end=$(date +%s)
  log "Built ${output} in $((target_end - target_start))s"
}

build_target linux amd64 linux x64 ""
build_target linux arm64 linux arm64 ""
build_target darwin amd64 darwin x64 ""
build_target darwin arm64 darwin arm64 ""
build_target windows amd64 win32 x64 ".exe"

log "Done. Assets in ${OUT_DIR}"

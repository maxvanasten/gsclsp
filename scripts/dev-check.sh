#!/usr/bin/env bash
set -euo pipefail

echo "[check] go vet ./..."
go vet ./...

echo "[check] go test ./..."
go test ./...

echo "[check] done"

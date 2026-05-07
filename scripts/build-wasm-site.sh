#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/web-dist"

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

cp -R "${ROOT_DIR}/web/." "${OUT_DIR}/"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "${OUT_DIR}/wasm_exec.js"

GOOS=js GOARCH=wasm go build \
  -trimpath \
  -ldflags="-s -w -X main.Version=${VERSION:-dev} -X main.Commit=${COMMIT:-local}" \
  -o "${OUT_DIR}/murmur.wasm" \
  ./cmd/wasm

if [ -d "${ROOT_DIR}/assets" ]; then
  cp -R "${ROOT_DIR}/assets" "${OUT_DIR}/assets"
fi

echo "WASM site bundle created at ${OUT_DIR}"

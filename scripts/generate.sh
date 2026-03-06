#!/usr/bin/env bash
# generate.sh — Generates Go code from proto definitions using buf.
#
# Prerequisites (install once):
#   go install github.com/bufbuild/buf/cmd/buf@latest
#
# Usage: ./scripts/generate.sh  (or: make generate)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${REPO_ROOT}/gen/go"

echo "▶ Cleaning generated code..."
find "${OUT_DIR}" -name "*.go" -type f -delete

echo "▶ Updating buf dependencies..."
cd "${REPO_ROOT}"
buf dep update

echo "▶ Generating Go + gRPC + gateway code from protos..."
buf generate

echo "✔ Code generation complete. Output: ${OUT_DIR}"

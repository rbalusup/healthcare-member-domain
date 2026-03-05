#!/usr/bin/env bash
# generate.sh — Runs protoc to generate Go code from proto definitions.
#
# Prerequisites (install once):
#   brew install protobuf
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#   go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
#   go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
#
# Usage: ./scripts/generate.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${REPO_ROOT}/api"
OUT_DIR="${REPO_ROOT}/gen/go"

echo "▶ Cleaning generated code..."
find "${OUT_DIR}" -name "*.go" -type f -delete

echo "▶ Generating Go + gRPC + gateway code from protos..."
find "${PROTO_DIR}" -name "*.proto" | while read -r proto_file; do
  relative_dir=$(dirname "${proto_file#"${REPO_ROOT}/"}")
  echo "  → ${proto_file}"
  protoc \
    --proto_path="${REPO_ROOT}/api" \
    --proto_path="${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway/v2@v2.20.0/third_party/googleapis" \
    --proto_path="${GOPATH}/pkg/mod/github.com/googleapis/googleapis@latest" \
    --go_out="${OUT_DIR}" \
    --go_opt=paths=source_relative \
    --go-grpc_out="${OUT_DIR}" \
    --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out="${OUT_DIR}" \
    --grpc-gateway_opt=paths=source_relative \
    "${proto_file}"
done

echo "✔ Code generation complete. Output: ${OUT_DIR}"

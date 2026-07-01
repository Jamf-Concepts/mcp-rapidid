#!/usr/bin/env bash
# Copyright 2026, Jamf Software LLC
# Compiles mcp-rapidid into build/package/mcpb/server/ for MCPB packaging.
# Usage:
#   ./scripts/build-bundle.sh                  # current OS/arch → server/mcp-rapidid
#   ./scripts/build-bundle.sh --all-platforms  # cross-compile all platforms
#
# After building:
#   cd build/package/mcpb/
#   mcpb validate
#   mcpb pack

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVER_DIR="${REPO_ROOT}/build/package/mcpb/server"
ENTRY="${REPO_ROOT}/cmd/mcp-rapidid"
BIN="mcp-rapidid"

mkdir -p "${SERVER_DIR}"

build_for() {
  local goos="$1" goarch="$2" output="$3"
  echo "Building ${goos}/${goarch} -> ${output}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "${SERVER_DIR}/${output}" "${ENTRY}"
}

if [[ "${1:-}" == "--all-platforms" ]]; then
  build_for darwin  arm64  "${BIN}-darwin-arm64"
  build_for darwin  amd64  "${BIN}-darwin-amd64"
  build_for linux   amd64  "${BIN}-linux-amd64"
  build_for linux   arm64  "${BIN}-linux-arm64"
  build_for windows amd64  "${BIN}-windows-amd64.exe"
  echo ""
  echo "Multi-platform build complete. Update manifest.json 'command' for each platform variant."
else
  build_for "$(go env GOOS)" "$(go env GOARCH)" "${BIN}"
  echo ""
  echo "Build complete: ${SERVER_DIR}/${BIN}"
fi

echo ""
echo "Next: cd build/package/mcpb/ && mcpb validate && mcpb pack"

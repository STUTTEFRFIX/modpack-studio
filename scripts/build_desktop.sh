#!/usr/bin/env bash
# Build the Windows EXE and a Linux test binary.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
mkdir -p dist

if [ ! -f internal/webui/graph.html ]; then
  echo "missing internal/webui/graph.html; run scripts/build_demo.sh first" >&2
  exit 1
fi

go test ./...

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -trimpath -ldflags "-s -w" \
  -o dist/ModpackStudio.exe ./cmd/modpack-studio-desktop

GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -trimpath -ldflags "-s -w" \
  -o dist/modpack-studio-desktop-linux ./cmd/modpack-studio-desktop

echo "EXE:   $ROOT/dist/ModpackStudio.exe"
echo "Linux: $ROOT/dist/modpack-studio-desktop-linux"

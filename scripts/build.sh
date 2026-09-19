#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 "${GO_BIN:-go}" build \
  -trimpath -ldflags='-s -w -H=windowsgui' \
  -o dist/DialogueForge.exe ./cmd/dialogueforge
echo 'Built dist/DialogueForge.exe'

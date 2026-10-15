#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  target_os=${target%/*}
  target_arch=${target#*/}
  suffix=""
  if [ "$target_os" = windows ]; then suffix=.exe; fi
  dest="dist/openvibecoding-${target_os}-${target_arch}"
  mkdir -p "$dest"
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -o "$dest/ovc$suffix" ./cmd/ovc
done

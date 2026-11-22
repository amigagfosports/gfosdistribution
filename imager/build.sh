#!/bin/sh
# One program for each system, in dist/. No C compiler needed.
set -e
cd "$(dirname "$0")"; mkdir -p dist
for t in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
  os=${t%/*}; arch=${t#*/}; ext=; [ "$os" = windows ] && ext=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "dist/gfos-imager-$os-$arch$ext" .
  echo "dist/gfos-imager-$os-$arch$ext"
done

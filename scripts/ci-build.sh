#!/bin/bash
set -e

VERSION="${1:-dev}"

PLATFORMS=(
  "linux:amd64"
  "linux:arm64"
  "linux:arm"
  "darwin:amd64"
  "darwin:arm64"
  "windows:amd64"
)

BINARIES=("sshifu" "sshifu-server" "sshifu-trust")

mkdir -p dist

for platform in "${PLATFORMS[@]}"; do
  OS="${platform%%:*}"
  ARCH="${platform##*:}"

  for binary in "${BINARIES[@]}"; do
    echo "Building ${binary} for ${OS}/${ARCH}..."

    EXT=""
    if [ "$OS" = "windows" ]; then
      EXT=".exe"
    fi

    # Use static linking for Linux to support musl-based systems (Alpine)
    CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go build -ldflags "-X main.version=$VERSION" -o "dist/${binary}-${OS}-${ARCH}${EXT}" "./cmd/${binary}"
    
    if [ "$OS" = "windows" ]; then
      cd dist
      7z a -tzip "${binary}-${OS}-${ARCH}.zip" "${binary}-${OS}-${ARCH}${EXT}"
      rm "${binary}-${OS}-${ARCH}${EXT}"
      cd ..
    else
      cd dist
      tar -czvf "${binary}-${OS}-${ARCH}.tar.gz" "${binary}-${OS}-${ARCH}${EXT}"
      rm "${binary}-${OS}-${ARCH}${EXT}"
      cd ..
    fi
  done
done

echo "Build complete! Artifacts in dist/"
ls -la dist/

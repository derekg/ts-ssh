#!/bin/bash
set -e

VERSION="v0.7.0"
OUTPUT_DIR="releases/v0.7.0"

# Build function
build() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="ts-ssh-${VERSION}-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    echo "Building $OUTPUT_NAME..."
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.version=${VERSION}" \
        -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
        .
    
    # Create checksum
    cd "${OUTPUT_DIR}"
    sha256sum "${OUTPUT_NAME}" > "${OUTPUT_NAME}.sha256"
    cd - > /dev/null
    
    echo "✓ Built ${OUTPUT_NAME}"
}

# Build for all platforms
echo "Building ts-ssh ${VERSION} for all platforms..."

# Linux
build linux amd64
build linux arm64

# macOS
build darwin amd64
build darwin arm64

# Windows
build windows amd64
build windows arm64

# BSD
build freebsd amd64
build openbsd amd64

echo ""
echo "Build complete! Artifacts in ${OUTPUT_DIR}/"
ls -lh ${OUTPUT_DIR}/

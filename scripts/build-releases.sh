#!/usr/bin/env bash
set -e

# ==============================================================================
# Bush Release Builder
# Packages standalone, statically linked tarballs for all supported platforms.
# ==============================================================================

VERSION="${1:-${VERSION}}"
if [ -z "$VERSION" ]; then
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v2.8.3")
fi

case "$VERSION" in
    v*) ;;
    *) VERSION="v$VERSION" ;;
esac

DIST_DIR="dist"
mkdir -p "$DIST_DIR"

echo "============================================================"
echo " Packaging Bush Release ${VERSION}"
echo "============================================================"

TARGETS=(
    "linux amd64"
    "linux arm64"
    "linux arm 7"
    "darwin amd64"
    "darwin arm64"
)

for target in "${TARGETS[@]}"; do
    read -r OS ARCH ARM_VER <<< "$target"

    LABEL_ARCH="$ARCH"
    if [ "$ARCH" = "arm" ] && [ "$ARM_VER" = "7" ]; then
        LABEL_ARCH="armv7"
    fi

    BUILD_DIR="build-${OS}-${LABEL_ARCH}"
    TARBALL="bush-${VERSION}-${OS}-${LABEL_ARCH}.tar.gz"

    echo "==> Compiling static binary for ${OS}/${LABEL_ARCH}..."
    mkdir -p "$BUILD_DIR"

    if [ -n "$ARM_VER" ]; then
        CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" GOARM="$ARM_VER" \
            go build -ldflags="-s -w -X main.Version=${VERSION}" -o "${BUILD_DIR}/bush" main.go
    else
        CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" \
            go build -ldflags="-s -w -X main.Version=${VERSION}" -o "${BUILD_DIR}/bush" main.go
    fi

    cp README.md LICENSE "$BUILD_DIR/"
    tar -czf "${DIST_DIR}/${TARBALL}" -C "$BUILD_DIR" bush README.md LICENSE
    rm -rf "$BUILD_DIR"

    echo "    Created: ${DIST_DIR}/${TARBALL}"
done

echo "==> Calculating SHA-256 checksums..."
cd "$DIST_DIR"
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum bush-*.tar.gz > SHA256SUMS
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 bush-*.tar.gz > SHA256SUMS
fi
cat SHA256SUMS
cd ..

echo "============================================================"
echo " Release artifacts generated in ${DIST_DIR}/"
echo "============================================================"

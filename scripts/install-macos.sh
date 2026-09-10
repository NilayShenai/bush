#!/usr/bin/env bash
# McBush: Automatic macOS installer script
set -e

echo "=== Installing McBush for macOS ==="

# 1. Determine architecture
ARCH=$(uname -m)
echo "[1/4] Detected architecture: ${ARCH}"

# 2. Compile binary
echo "[2/4] Building bush binary..."
go build -ldflags="-s -w" -o bush main.go

# 3. Choose install destination
TARGET_DIR="/usr/local/bin"
if [ "${ARCH}" = "arm64" ] && [ -d "/opt/homebrew/bin" ]; then
    TARGET_DIR="/usr/local/bin"
fi

echo "[3/4] Installing to ${TARGET_DIR}/bush..."
sudo mkdir -p "${TARGET_DIR}"
sudo cp bush "${TARGET_DIR}/bush"
sudo chmod 755 "${TARGET_DIR}/bush"

# 4. Add to /etc/shells if not already present
echo "[4/4] Registering in /etc/shells..."
if ! grep -qxF "${TARGET_DIR}/bush" /etc/shells; then
    echo "${TARGET_DIR}/bush" | sudo tee -a /etc/shells > /dev/null
fi

echo ""
echo "=== McBush Installed Successfully! ==="
echo "To test run:"
echo "  ${TARGET_DIR}/bush"
echo ""
echo "To set as your default login shell:"
echo "  chsh -s ${TARGET_DIR}/bush"
echo ""

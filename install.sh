#!/usr/bin/env bash
set -e

# ==============================================================================
# Bush Shell Universal Installer
# "A shell for people of refined taste."
# Repository: https://github.com/NilayShenai/bush
# ==============================================================================

REPO="NilayShenai/bush"
DEFAULT_VERSION="v2.8.2"

# Color Codes (Bush Lavender/Pastel Palette)
ESC="\033["
RESET="${ESC}0m"
BOLD="${ESC}1m"
ACCENT="${ESC}38;2;180;190;254m"
MAUVE="${ESC}38;2;203;166;247m"
MINT="${ESC}38;2;148;226;213m"
PEACH="${ESC}38;2;250;179;135m"
RED="${ESC}38;2;243;139;168m"
GRAY="${ESC}38;2;108;112;134m"

info() {
    printf "${BOLD}${ACCENT}[bush-install]${RESET} %s\n" "$1"
}

success() {
    printf "${BOLD}${MINT}[bush-install]${RESET} %s\n" "$1"
}

warn() {
    printf "${BOLD}${PEACH}[bush-install]${RESET} %s\n" "$1"
}

error() {
    printf "${BOLD}${RED}[bush-install] ERROR:${RESET} %s\n" "$1" >&2
    exit 1
}

# 1. Check required downloader (curl or wget)
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget"
else
    error "Neither 'curl' nor 'wget' was found on your system. Please install one to continue."
fi

download_file() {
    local url="$1"
    local dest="$2"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url" -o "$dest"
    else
        wget -q "$url" -O "$dest"
    fi
}

download_stdout() {
    local url="$1"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url"
    else
        wget -qO- "$url"
    fi
}

# 2. Check for tar
if ! command -v tar >/dev/null 2>&1; then
    error "'tar' is required to extract the release archive."
fi

# 3. Detect OS
OS_RAW="$(uname -s)"
case "$OS_RAW" in
    Linux)
        OS="linux"
        ;;
    Darwin)
        OS="darwin"
        ;;
    *)
        error "Unsupported operating system: $OS_RAW. Bush currently supports Linux and macOS."
        ;;
esac

# 4. Detect CPU Architecture
ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l|armv7)
        ARCH="armv7"
        ;;
    *)
        error "Unsupported CPU architecture: $ARCH_RAW. Supported architectures: amd64, arm64, armv7."
        ;;
esac

# 5. Resolve Version
VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
    info "Resolving latest release from GitHub..."
    LATEST_TAG=$(download_stdout "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
    if [ -n "$LATEST_TAG" ]; then
        VERSION="$LATEST_TAG"
    else
        warn "Could not fetch latest release tag via GitHub API (possibly rate-limited). Falling back to default: ${DEFAULT_VERSION}"
        VERSION="$DEFAULT_VERSION"
    fi
fi

# Ensure version has 'v' prefix
case "$VERSION" in
    v*) ;;
    *) VERSION="v$VERSION" ;;
esac

info "Installing Bush ${BOLD}${VERSION}${RESET} for ${BOLD}${OS}/${ARCH}${RESET}..."

# 6. Temporary working directory
TMP_DIR="$(mktemp -d -t bush-install-XXXXXX)"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

TARBALL="bush-${VERSION}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/SHA256SUMS"

info "Downloading ${DOWNLOAD_URL}..."
if ! download_file "$DOWNLOAD_URL" "${TMP_DIR}/${TARBALL}"; then
    error "Failed to download ${TARBALL}. Please verify that release ${VERSION} exists at https://github.com/${REPO}/releases."
fi

# Optional: Verify SHA256 Checksum if available
if download_file "$CHECKSUMS_URL" "${TMP_DIR}/SHA256SUMS" 2>/dev/null; then
    info "Verifying SHA256 integrity checksum..."
    EXPECTED_HASH=$(grep "${TARBALL}" "${TMP_DIR}/SHA256SUMS" | awk '{print $1}' || true)
    if [ -n "$EXPECTED_HASH" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL_HASH=$(sha256sum "${TMP_DIR}/${TARBALL}" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL_HASH=$(shasum -a 256 "${TMP_DIR}/${TARBALL}" | awk '{print $1}')
        else
            ACTUAL_HASH=""
        fi

        if [ -n "$ACTUAL_HASH" ]; then
            if [ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]; then
                error "Checksum verification failed! Expected ${EXPECTED_HASH}, got ${ACTUAL_HASH}."
            fi
            success "Checksum verified successfully (${ACTUAL_HASH:0:12}...)."
        fi
    fi
fi

# 7. Extract archive
info "Extracting archive..."
tar -xzf "${TMP_DIR}/${TARBALL}" -C "$TMP_DIR"

if [ ! -f "${TMP_DIR}/bush" ]; then
    error "Archive did not contain the 'bush' binary."
fi

# 8. Determine destination directory
INSTALL_DIR="${INSTALL_DIR:-}"

if [ -z "$INSTALL_DIR" ]; then
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO=false
    elif command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO=true
    else
        INSTALL_DIR="${HOME}/.local/bin"
        USE_SUDO=false
    fi
else
    USE_SUDO=false
fi

mkdir -p "$INSTALL_DIR" 2>/dev/null || true

TARGET_PATH="${INSTALL_DIR}/bush"
info "Installing binary to ${TARGET_PATH}..."

if [ "$USE_SUDO" = true ]; then
    warn "/usr/local/bin is not user-writable. Escalating with sudo..."
    sudo install -m 755 "${TMP_DIR}/bush" "$TARGET_PATH"
else
    install -m 755 "${TMP_DIR}/bush" "$TARGET_PATH" 2>/dev/null || cp "${TMP_DIR}/bush" "$TARGET_PATH"
    chmod 755 "$TARGET_PATH"
fi

# 9. Final verification and instructions
printf "\n"
printf "${BOLD}${ACCENT}+-------------------------------------------------------------+${RESET}\n"
printf "${BOLD}${ACCENT}|                    BUSH SHELL INSTALLED                     |${RESET}\n"
printf "${BOLD}${MAUVE}|              FOR BUSH LOVERS, BY BUSH LOVERS                |${RESET}\n"
printf "${BOLD}${ACCENT}+-------------------------------------------------------------+${RESET}\n"
printf "  ${BOLD}%s${RESET} is now installed at ${BOLD}${MINT}%s${RESET}\n\n" "Bush" "$TARGET_PATH"

if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    warn "'${INSTALL_DIR}' is not currently in your \$PATH."
    if [ -f "$HOME/.bashrc" ] && ! grep -q "$INSTALL_DIR" "$HOME/.bashrc"; then
        printf "\nexport PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR" >> "$HOME/.bashrc"
        info "Added '${INSTALL_DIR}' to ~/.bashrc"
    fi
    if [ -f "$HOME/.zshrc" ] && ! grep -q "$INSTALL_DIR" "$HOME/.zshrc"; then
        printf "\nexport PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR" >> "$HOME/.zshrc"
        info "Added '${INSTALL_DIR}' to ~/.zshrc"
    fi
    printf "To use bush in your current terminal tab, run:\n"
    printf "${BOLD}export PATH=\"%s:\$PATH\"${RESET}\n\n" "$INSTALL_DIR"
fi

printf "${BOLD}${PEACH}Next Steps:${RESET}\n\n"
printf "1. Start Bush right now:\n"
printf "${BOLD}${MINT}%s${RESET}\n\n" "$TARGET_PATH"
printf "2. Register Bush as a recognized login shell (optional):\n"
printf "${BOLD}echo \"%s\" | sudo tee -a /etc/shells${RESET}\n" "$TARGET_PATH"
printf "${BOLD}chsh -s \"%s\"${RESET}\n\n" "$TARGET_PATH"
printf "3. To configure themes, aliases, and prompts:\n"
printf "Type ${BOLD}${ACCENT}config${RESET} inside Bush or edit ${BOLD}~/.config/bush/config.toml${RESET}\n\n"

success "Installation complete! Enjoy Bush."

#!/bin/sh
set -e

# git-next installer
# Usage: curl -sfL https://raw.githubusercontent.com/VectorSophie/git-next/main/scripts/install.sh | sh
# Pin a version: VERSION=v0.3.3 curl -sfL ... | sh

REPO="VectorSophie/git-next"
BINARY_NAME="git-next"

# ── helpers ──────────────────────────────────────────────────────────────────

info()  { printf '  \033[34m%s\033[0m\n' "$*"; }
ok()    { printf '  \033[32m✓\033[0m %s\n' "$*"; }
err()   { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }
need()  { command -v "$1" >/dev/null 2>&1 || err "required tool not found: $1"; }

# ── requirements ─────────────────────────────────────────────────────────────

need curl

# ── platform detection ────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
ARM=""

case "$OS" in
    linux|darwin) ;;
    *) err "unsupported OS: $OS (Windows users: download from https://github.com/$REPO/releases)" ;;
esac

case "$ARCH" in
    x86_64)         ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    armv7l)         ARCH="arm"; ARM="v7" ;;
    armv6l)         ARCH="arm"; ARM="v6" ;;
    *)              err "unsupported architecture: $ARCH" ;;
esac

# ── version resolution ────────────────────────────────────────────────────────

if [ -z "$VERSION" ]; then
    info "Fetching latest release..."
    VERSION=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" \
        | grep '"tag_name":' \
        | sed -E 's/.*"([^"]+)".*/\1/')
    [ -n "$VERSION" ] || err "failed to fetch latest version from GitHub API"
fi

VERSION_BARE="${VERSION#v}"
info "Version: $VERSION"

# ── install directory ─────────────────────────────────────────────────────────

# Prefer /usr/local/bin; fall back to ~/.local/bin (no sudo required).
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
    NEED_SUDO=false
elif [ "$(id -u)" = "0" ]; then
    INSTALL_DIR="/usr/local/bin"
    NEED_SUDO=false
elif command -v sudo >/dev/null 2>&1; then
    INSTALL_DIR="/usr/local/bin"
    NEED_SUDO=true
else
    INSTALL_DIR="$HOME/.local/bin"
    NEED_SUDO=false
    mkdir -p "$INSTALL_DIR"
fi

info "Install directory: $INSTALL_DIR"

# ── download ──────────────────────────────────────────────────────────────────

ARCHIVE="${BINARY_NAME}_${VERSION_BARE}_${OS}_${ARCH}${ARM}.tar.gz"
BASE_URL="https://github.com/$REPO/releases/download/$VERSION"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading $ARCHIVE..."
curl -sfL "$BASE_URL/$ARCHIVE" -o "$TMP_DIR/$ARCHIVE" \
    || err "download failed: $BASE_URL/$ARCHIVE"

# ── checksum verification ─────────────────────────────────────────────────────

CHECKSUMS_FILE="checksums.txt"
info "Verifying checksum..."
curl -sfL "$BASE_URL/$CHECKSUMS_FILE" -o "$TMP_DIR/$CHECKSUMS_FILE" \
    || err "failed to download checksums from $BASE_URL/$CHECKSUMS_FILE"

# sha256sum is standard on Linux; shasum ships on macOS.
if command -v sha256sum >/dev/null 2>&1; then
    cd "$TMP_DIR" && grep "$ARCHIVE" "$CHECKSUMS_FILE" | sha256sum -c - >/dev/null 2>&1 \
        || err "checksum verification failed — download may be corrupt or tampered"
elif command -v shasum >/dev/null 2>&1; then
    cd "$TMP_DIR" && grep "$ARCHIVE" "$CHECKSUMS_FILE" | shasum -a 256 -c - >/dev/null 2>&1 \
        || err "checksum verification failed — download may be corrupt or tampered"
else
    info "WARNING: sha256sum/shasum not found — skipping checksum verification"
fi

ok "Checksum verified"

# ── extract ───────────────────────────────────────────────────────────────────

cd "$TMP_DIR"
tar -xzf "$ARCHIVE"

# ── install ───────────────────────────────────────────────────────────────────

if [ "$NEED_SUDO" = "true" ]; then
    info "Installing to $INSTALL_DIR (sudo required)..."
    sudo install -m 755 "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    info "Installing to $INSTALL_DIR..."
    install -m 755 "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

# ── verify ────────────────────────────────────────────────────────────────────

INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY_NAME" --version 2>&1 | head -1)

echo ""
ok "$BINARY_NAME installed successfully!"
info "$INSTALLED_VERSION"
echo ""

# Warn if the install dir isn't on PATH.
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        printf '  \033[33m!\033[0m %s is not in your PATH.\n' "$INSTALL_DIR"
        printf '    Add this to your shell profile:\n'
        printf '    export PATH="%s:$PATH"\n' "$INSTALL_DIR"
        echo ""
        ;;
esac

info "Run 'git-next' in a git repository to get started."
info "Run 'git-next --help' to see all options."

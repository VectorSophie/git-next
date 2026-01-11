#!/bin/sh
set -e

# git-next installer
# Usage: curl -sfL https://raw.githubusercontent.com/yourusername/git-next/main/scripts/install.sh | sh

REPO="yourusername/git-next"
BINARY_NAME="git-next"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case $OS in
    linux|darwin)
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Get latest release version
echo "Fetching latest release..."
LATEST_VERSION=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "Failed to fetch latest version"
    exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Construct download URL
if [ "$OS" = "linux" ] || [ "$OS" = "darwin" ]; then
    ARCHIVE_NAME="${BINARY_NAME}_${LATEST_VERSION#v}_${OS}_${ARCH}.tar.gz"
else
    ARCHIVE_NAME="${BINARY_NAME}_${LATEST_VERSION#v}_${OS}_${ARCH}.zip"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$ARCHIVE_NAME"

echo "Downloading $ARCHIVE_NAME..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

if ! curl -sfLO "$DOWNLOAD_URL"; then
    echo "Failed to download from $DOWNLOAD_URL"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Extract archive
echo "Extracting..."
if [ "$OS" = "linux" ] || [ "$OS" = "darwin" ]; then
    tar -xzf "$ARCHIVE_NAME"
else
    unzip -q "$ARCHIVE_NAME"
fi

# Install binary
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Need sudo to install to $INSTALL_DIR"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

# Cleanup
cd -
rm -rf "$TMP_DIR"

echo ""
echo "✓ $BINARY_NAME installed successfully!"
echo ""
echo "Run '$BINARY_NAME --version' to verify installation"
echo "Run '$BINARY_NAME' in a git repository to get started"

#!/bin/sh
set -e

REPO="ariffrahimin/sonarsweep"
BIN_NAME="sonarsweep"
INSTALL_DIR="/usr/local/bin"

echo "==> Installing $BIN_NAME..."

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "$OS" != "darwin" ] && [ "$OS" != "linux" ]; then
    echo "Error: Unsupported OS '$OS'"
    exit 1
fi

if [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "amd64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
else
    echo "Error: Unsupported architecture '$ARCH'"
    exit 1
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/${BIN_NAME}-${OS}-${ARCH}.tar.gz"
TEMP_DIR=$(mktemp -d)

echo "==> Downloading from $DOWNLOAD_URL..."
curl -L -s -o "$TEMP_DIR/${BIN_NAME}.tar.gz" "$DOWNLOAD_URL"

echo "==> Extracting..."
tar -xzf "$TEMP_DIR/${BIN_NAME}.tar.gz" -C "$TEMP_DIR"

echo "==> Installing to $INSTALL_DIR..."
if [ ! -d "$INSTALL_DIR" ]; then
    sudo mkdir -p "$INSTALL_DIR"
fi

if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_DIR/$BIN_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BIN_NAME"
else
    echo "==> Requires sudo privileges to write to $INSTALL_DIR"
    sudo mv "$TEMP_DIR/$BIN_NAME" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/$BIN_NAME"
fi

rm -rf "$TEMP_DIR"

echo "==> Installation complete! You can now run '$BIN_NAME'."

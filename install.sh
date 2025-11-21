#!/bin/sh
set -e

REPO="edwardoboh/melinolb"
BINARY_NAME="melinolb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case $OS in
    linux*) OS="linux" ;;
    darwin*) OS="darwin" ;;
    msys*|mingw*|cygwin*) OS="windows" ;;
    *) echo "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) echo "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# Get latest version
echo "${YELLOW}Fetching latest version...${NC}"
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "${RED}Failed to fetch latest version${NC}"
    exit 1
fi

echo "${GREEN}Latest version: $VERSION${NC}"

# Construct download URL
FILE_EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    FILE_EXT="zip"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.${FILE_EXT}"

# Download and extract
echo "${YELLOW}Downloading $BINARY_NAME...${NC}"
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

if command -v wget > /dev/null; then
    wget -q "$DOWNLOAD_URL" -O "archive.${FILE_EXT}"
else
    curl -sL "$DOWNLOAD_URL" -o "archive.${FILE_EXT}"
fi

# Extract based on file type
if [ "$FILE_EXT" = "zip" ]; then
    unzip -q "archive.${FILE_EXT}"
else
    tar xzf "archive.${FILE_EXT}"
fi

# Determine install location
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
    mv "$BINARY_NAME" "$INSTALL_DIR/"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
    mv "$BINARY_NAME" "$INSTALL_DIR/"
    echo "${YELLOW}Add $INSTALL_DIR to your PATH: export PATH=\$PATH:$INSTALL_DIR${NC}"
fi

# Cleanup
cd - > /dev/null
rm -rf "$TMP_DIR"

echo "${GREEN}✓ $BINARY_NAME $VERSION installed to $INSTALL_DIR${NC}"
echo "${GREEN}Run '$BINARY_NAME --version' to verify installation${NC}"
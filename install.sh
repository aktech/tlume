#!/bin/sh
# tlume - Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/aktech/tlume/main/install.sh | sh

set -e

# Text colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Print banner
echo "${BLUE}${BOLD}"
echo " _______ _                          "
echo "|__   __| |                         "
echo "   | |  | |    _   _ _ __ ___   ___ "
echo "   | |  | |   | | | | '_ \` _ \\ / _ \\"
echo "   | |  | |___| |_| | | | | | |  __/"
echo "   |_|  |______\\__,_|_| |_| |_|\\___|"
echo "${NC}"
echo "${BOLD}Tart to Lume VM Converter - Installer${NC}"
echo ""

# Check for curl or wget
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_CMD="curl -L -s"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_CMD="wget -q -O-"
else
    echo "${RED}Error: Neither curl nor wget found. Please install one of them and try again.${NC}"
    exit 1
fi

# Get system information
get_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        i386|i686)
            echo "386"
            ;;
        *)
            echo "${RED}Unsupported architecture: $ARCH${NC}" >&2
            exit 1
            ;;
    esac
}

get_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case $OS in
        linux)
            echo "linux"
            ;;
        darwin)
            echo "darwin"
            ;;
        msys*|mingw*|cygwin*)
            echo "windows"
            ;;
        *)
            echo "${RED}Unsupported OS: $OS${NC}" >&2
            exit 1
            ;;
    esac
}

# Detect user shell profile file
detect_profile() {
    SHELL_NAME=$(basename "$SHELL")

    if [ "$OS" = "darwin" ]; then
        # macOS (Darwin) shell detection
        if [ "$SHELL_NAME" = "zsh" ]; then
            if [ -f "$HOME/.zshrc" ]; then
                echo "$HOME/.zshrc"
            else
                echo "$HOME/.zprofile"
            fi
        elif [ "$SHELL_NAME" = "bash" ]; then
            if [ -f "$HOME/.bash_profile" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
        else
            echo "$HOME/.profile"
        fi
    else
        # Linux and other systems
        if [ "$SHELL_NAME" = "zsh" ]; then
            echo "$HOME/.zshrc"
        elif [ "$SHELL_NAME" = "bash" ]; then
            echo "$HOME/.bashrc"
        else
            echo "$HOME/.profile"
        fi
    fi
}

OS=$(get_os)
ARCH=$(get_arch)

# Create temp directory first - before we use it
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Get the latest version
echo "${BLUE}→ ${NC}Detecting latest version..."
VERSION=$($DOWNLOAD_CMD https://api.github.com/repos/aktech/tlume/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "${RED}Error: Could not determine the latest version. Please check your internet connection.${NC}"
    exit 1
fi

echo "${GREEN}✓ ${NC}Latest version: ${BLUE}${VERSION}${NC}"

# Create the download URL
RELEASE_URL="https://github.com/aktech/tlume/releases/download/${VERSION}"

# Determine the package extension
if [ "$OS" = "windows" ]; then
    EXT="zip"
    BINARY_NAME="tlume.exe"
else
    EXT="tar.gz"
    BINARY_NAME="tlume"
fi

if [ "$ARCH" = "amd64" ]; then
    ARCH_NAME="x86_64"
elif [ "$ARCH" = "386" ]; then
    ARCH_NAME="i386"
else
    ARCH_NAME="$ARCH"
fi

# Create download file name - without using ${VAR^} which isn't supported in all shells
OS_PROPER="$OS"
# Manually capitalize first letter if needed
if [ "$OS" = "linux" ]; then
    OS_PROPER="Linux"
elif [ "$OS" = "darwin" ]; then
    OS_PROPER="Darwin"
elif [ "$OS" = "windows" ]; then
    OS_PROPER="Windows"
fi

PACKAGE_NAME="tlume_${OS_PROPER}_${ARCH_NAME}"
DOWNLOAD_URL="${RELEASE_URL}/${PACKAGE_NAME}.${EXT}"
PACKAGE_PATH="${TMP_DIR}/tlume.${EXT}"

# Download the package
echo "${BLUE}→ ${NC}Downloading tlume ${VERSION} for ${OS}/${ARCH}..."
echo "${BLUE}→ ${NC}Download URL: ${DOWNLOAD_URL}"

if command -v curl >/dev/null 2>&1; then
    curl -L -s -o "$PACKAGE_PATH" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$PACKAGE_PATH" "$DOWNLOAD_URL"
else
    echo "${RED}Error: Neither curl nor wget found. Please install one of them and try again.${NC}"
    exit 1
fi

if [ ! -f "$PACKAGE_PATH" ] || [ ! -s "$PACKAGE_PATH" ]; then
    echo "${RED}Error: Download failed. Please check your internet connection and try again.${NC}"
    exit 1
fi

echo "${GREEN}✓ ${NC}Download complete"

# Extract the package
echo "${BLUE}→ ${NC}Extracting package..."
if [ "$OS" = "windows" ]; then
    if command -v unzip >/dev/null 2>&1; then
        unzip -q "$PACKAGE_PATH" -d "$TMP_DIR"
    else
        echo "${RED}Error: unzip command not found. Please install unzip and try again.${NC}"
        exit 1
    fi
else
    tar -xzf "$PACKAGE_PATH" -C "$TMP_DIR"
fi

BINARY_PATH="$TMP_DIR/$BINARY_NAME"
if [ ! -f "$BINARY_PATH" ]; then
    # Try to find the binary in subdirectories
    BINARY_PATH=$(find "$TMP_DIR" -name "$BINARY_NAME" -type f | head -n 1)
    if [ -z "$BINARY_PATH" ]; then
        echo "${RED}Error: Could not find tlume binary in the downloaded package.${NC}"
        exit 1
    fi
fi

chmod +x "$BINARY_PATH"

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    # Use home directory if user doesn't have write permissions to /usr/local/bin
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"

    # Add to PATH if needed by updating the appropriate profile file
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        PROFILE_FILE=$(detect_profile)
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$PROFILE_FILE"
        echo "${YELLOW}Note: Added $INSTALL_DIR to your PATH in $PROFILE_FILE${NC}"
        echo "${YELLOW}To use tlume in the current shell, run: export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
    fi
fi

# Install the binary
echo "${BLUE}→ ${NC}Installing to $INSTALL_DIR..."
cp "$BINARY_PATH" "$INSTALL_DIR/tlume"

# Verify installation
if [ -f "$INSTALL_DIR/tlume" ]; then
    echo "${GREEN}✓ ${NC}tlume ${VERSION} has been installed successfully!"
    echo ""
    echo "${BLUE}To use tlume, run:${NC}"
    echo "  tlume <machine_name>"
    echo ""
    echo "${BLUE}Example:${NC}"
    echo "  tlume macos-ventura"
    echo ""
    echo "${BLUE}Example with options:${NC}"
    echo "  tlume -disk-size 100GB macos-ventura"
    echo ""
    echo "${BLUE}For help:${NC}"
    echo "  tlume -h"
    echo ""

    # Remind to source profile or restart shell if we modified it
    if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
        PROFILE_FILE=$(detect_profile)
        echo "${YELLOW}Remember to either:${NC}"
        echo "  1. Run 'source $PROFILE_FILE' in your current shell, or"
        echo "  2. Start a new terminal session"
        echo "${YELLOW}for the command to be available in your PATH.${NC}"
    fi
else
    echo "${RED}Installation seems to have failed. Please try again or install manually.${NC}"
    echo "You can download the binary directly from: $DOWNLOAD_URL"
    exit 1
fi

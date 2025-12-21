#!/bin/bash

# Universal install script - detects platform and installs accordingly

echo "Installing aerosync-service..."

# Detect OS and set paths
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Detected Linux, installing to /usr/local/bin..."
    BINARY_SRC="bin/aerosync-service"
    BINARY_DST="/usr/local/bin/aerosync-service"
    INSTALL_CMD="sudo cp $BINARY_SRC $BINARY_DST && sudo chmod +x $BINARY_DST"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    echo "Detected Windows, installing to ~/bin..."
    BINARY_SRC="bin/aerosync-service.exe"
    BINARY_DST="$HOME/bin/aerosync-service.exe"
    mkdir -p "$HOME/bin"
    INSTALL_CMD="cp $BINARY_SRC $BINARY_DST"
else
    echo "❌ Unsupported OS: $OSTYPE"
    exit 1
fi

# Check if binary exists
if [ ! -f "$BINARY_SRC" ]; then
    echo "❌ Binary not found: $BINARY_SRC"
    echo "Run ./build.sh first."
    exit 1
fi

# Install
eval "$INSTALL_CMD"

if [ $? -eq 0 ]; then
    echo "Installation successful!"
    echo "Binary installed to: $BINARY_DST"
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Run 'aerosync-service --help' to get started."
    else
        echo "Make sure ~/bin is in your PATH to run 'aerosync-service.exe --help'"
    fi
else
    echo "Installation failed"
    exit 1
fi

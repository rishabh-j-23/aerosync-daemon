#!/bin/bash

# Universal build script - detects platform and builds accordingly

echo "Building aerosync-service..."

# Create bin directory if it doesn't exist
mkdir -p bin

go mod tidy

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Detected Linux, building for Linux..."
    GOOS=linux GOARCH=amd64 go build -o bin/aerosync-service -v
    BINARY="bin/aerosync-service"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    echo "Detected Windows, building for Windows..."
    GOOS=windows GOARCH=amd64 go build -o bin/aerosync-service.exe -v
    BINARY="bin/aerosync-service.exe"
else
    echo "❌ Unsupported OS: $OSTYPE"
    exit 1
fi

if [ $? -eq 0 ]; then
    echo "Build successful: $BINARY"
else
    echo "Build failed"
    exit 1
fi

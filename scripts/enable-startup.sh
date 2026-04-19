#!/bin/bash

# Enable automatic startup for aerosync-service on Linux

echo "Enabling automatic startup for aerosync-service..."

# Check if binary exists
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    BINARY_PATH="/usr/local/bin/aerosync-service"
else
    echo "❌ This script is for Linux systems only"
    exit 1
fi

if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ Binary not found: $BINARY_PATH"
    echo "Run ./scripts/install.sh first."
    exit 1
fi

# Create systemd service file
SERVICE_FILE="/etc/systemd/system/aerosync.service"
SERVICE_CONTENT="[Unit]
Description=Aerosync Background Sync Service
After=network.target

[Service]
Type=simple
User=$USER
ExecStart=$BINARY_PATH start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target"

echo "Creating systemd service file..."
echo "$SERVICE_CONTENT" | sudo tee "$SERVICE_FILE" > /dev/null

if [ $? -eq 0 ]; then
    echo "Reloading systemd daemon..."
    sudo systemctl daemon-reload

    echo "Enabling service..."
    sudo systemctl enable aerosync.service

    echo "Starting service..."
    sudo systemctl start aerosync.service

    echo "✅ Automatic startup enabled successfully!"
    echo "Service status: $(sudo systemctl is-active aerosync.service)"
    echo "Run './scripts/disable-startup.sh' to disable automatic startup."
else
    echo "❌ Failed to create systemd service"
    exit 1
fi
#!/bin/bash

# Disable automatic startup for aerosync-service on Linux

echo "Disabling automatic startup for aerosync-service..."

if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "❌ This script is for Linux systems only"
    exit 1
fi

SERVICE_FILE="/etc/systemd/system/aerosync.service"

if [ ! -f "$SERVICE_FILE" ]; then
    echo "❌ Service file does not exist: $SERVICE_FILE"
    echo "Automatic startup is already disabled."
    exit 0
fi

echo "Stopping service..."
sudo systemctl stop aerosync.service

echo "Disabling service..."
sudo systemctl disable aerosync.service

echo "Removing service file..."
sudo rm "$SERVICE_FILE"

echo "Reloading systemd daemon..."
sudo systemctl daemon-reload

echo "✅ Automatic startup disabled successfully!"
echo "The service will no longer start automatically on boot."
echo "Run './scripts/enable-startup.sh' to re-enable automatic startup."
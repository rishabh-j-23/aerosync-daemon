# Makefile for Aerosync Service

.ONESHELL:

.PHONY: build install clean help create-task delete-task

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
BINARY_NAME=aerosync-service

# OS-specific settings
ifeq ($(OS),Windows_NT)
    BINARY_PATH=bin/$(BINARY_NAME).exe
    INSTALL_DIR=$(USERPROFILE)\bin
    INSTALL_CMD=if not exist $(INSTALL_DIR) mkdir $(INSTALL_DIR) && copy $(BINARY_PATH) $(INSTALL_DIR)\$(BINARY_NAME).exe
else
    BINARY_PATH=bin/$(BINARY_NAME)
    INSTALL_DIR=/usr/local/bin
    INSTALL_CMD=sudo cp $(BINARY_PATH) $(INSTALL_DIR)/$(BINARY_NAME) && sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
endif
# Add Darwin for macOS if needed

# Build target
build:
	@echo "Building $(BINARY_NAME)..."
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File scripts\build.ps1
else
	./scripts/build.sh
endif

# Install target
install: build
	@echo "Installing $(BINARY_NAME)..."
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File scripts\install.ps1
else
	./scripts/install.sh
endif

# Create task target
create-task:
	@echo "Creating startup task..."
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File scripts\enable-startup.ps1
else
	./scripts/enable-startup.sh
endif

# Delete task target
delete-task:
	@echo "Deleting startup task..."
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File scripts\disable-startup.ps1
else
	./scripts/disable-startup.sh
endif

# Clean target
clean:
	@echo "Cleaning build artifacts..."
	- rm -rf ./bin/
	- rm -f ~/.config/aerosync/metadata.db
ifeq ($(OS),Linux)
	- sudo systemctl stop aerosync.service 2>/dev/null || true
	- sudo systemctl disable aerosync.service 2>/dev/null || true
	- sudo rm -f /etc/systemd/system/aerosync.service
	- sudo systemctl daemon-reload 2>/dev/null || true
endif

# Help target
help:
	@echo "Available targets:"
	@echo "  build       - Build binary for current platform"
	@echo "  install     - Build and install binary"
	@echo "  create-task - Create startup task for automatic launch"
	@echo "  delete-task - Delete startup task"
	@echo "  clean       - Remove build artifacts"
	@echo "  help        - Show this help"

# Makefile for Aerosync Service

.ONESHELL:

.PHONY: build install clean help

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

# Clean target
clean:
	@echo "Cleaning build artifacts..."
	- rm -rf ./bin/
	- rm -f ~/.config/aerosync/metadata.db

# Help target
help:
	@echo "Available targets:"
	@echo "  build   - Build binary for current platform"
	@echo "  install - Build and install binary"
	@echo "  clean   - Remove build artifacts"
	@echo "  help    - Show this help"

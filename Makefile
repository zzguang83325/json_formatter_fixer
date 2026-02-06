# Makefile for Wails project

APP_NAME = Json-Formatter-Fixer
BUILD_DIR = build/bin

.PHONY: all help windows darwin linux clean install-deps

all: windows darwin linux
	@echo "All platforms built successfully."

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  windows      Build the application for Windows"
	@echo "  darwin       Build the application for macOS"
	@echo "  linux        Build the application for Linux"
	@echo "  all          Build for all platforms"
	@echo "  clean        Clean the build directory"
	@echo "  install-deps Install frontend dependencies"

# Build for Windows
windows:
	@echo "Building for Windows..."
	wails build -platform windows/amd64 -o $(APP_NAME).exe

# Build for macOS
darwin:
	@echo "Building for macOS..."
	wails build -platform darwin/universal

# Build for Linux
linux:
	@echo "Building for Linux..."
	wails build -platform linux/amd64

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)/*
	@echo "Cleaned."

# Install frontend dependencies
install-deps:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

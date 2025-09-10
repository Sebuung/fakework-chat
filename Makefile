# Makefile for fakework-chat

.PHONY: all release clean build-linux build-windows build-darwin

all: release

release: build-linux build-windows build-darwin

# Output directory
BUILD_DIR := build

# Source paths
SERVER_SRC := ./cmd/server
CLIENT_SRC := ./cmd/client

# Build for Linux
build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/server-linux-amd64 $(SERVER_SRC)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/fakeworkchat-linux-amd64 $(CLIENT_SRC)

# Build for Windows
build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)/windows-amd64
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/server-windows-amd64.exe $(SERVER_SRC)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/fakeworkchat-windows-amd64.exe $(CLIENT_SRC)

# Build for Darwin (macOS)
build-darwin:
	@echo "Building for Darwin (amd64)..."
	@mkdir -p $(BUILD_DIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/server-darwin-amd64 $(SERVER_SRC)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/fakeworkchat-darwin-amd64 $(CLIENT_SRC)
	@echo "Building for Darwin (arm64)..."
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/server-darwin-arm64 $(SERVER_SRC)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/fakeworkchat-darwin-arm64 $(CLIENT_SRC)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)/*

#!/bin/bash
# Shared build script for cross-platform binaries
# Used by both release workflow and CI workflow

set -e

echo "🔨 Building cross-platform binaries..."

# Get version from environment or git tag
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building version: ${VERSION}"

# Linux AMD64
echo "Building Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o buzz-linux-amd64 .

# Linux ARM64
echo "Building Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o buzz-linux-arm64 .

# macOS Intel
echo "Building macOS Intel..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o buzz-darwin-amd64 .

# macOS Apple Silicon
echo "Building macOS Apple Silicon..."
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o buzz-darwin-arm64 .

# Windows AMD64
echo "Building Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o buzz-windows-amd64.exe .

# Windows ARM64
echo "Building Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o buzz-windows-arm64.exe .

echo "✅ All builds completed successfully!"

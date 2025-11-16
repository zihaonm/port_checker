#!/bin/bash

set -e

echo "Building port-checker for Linux amd64..."
echo ""
echo "Note: This will embed .env and targets.txt into the binary"
echo ""

# Check if files exist
if [ ! -f ".env" ]; then
    echo "WARNING: .env file not found - binary will not have embedded credentials"
fi

if [ ! -f "targets.txt" ]; then
    echo "WARNING: targets.txt file not found - binary will not have embedded targets"
fi

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Build for Linux amd64
echo "Cross-compiling for Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o port-checker-linux

echo ""
echo "✓ Build successful!"
echo "Binary created: ./port-checker-linux"
echo ""
echo "Embedded files:"
if [ -f ".env" ]; then
    echo "  ✓ .env"
else
    echo "  ✗ .env (not found)"
fi
if [ -f "targets.txt" ]; then
    echo "  ✓ targets.txt"
else
    echo "  ✗ targets.txt (not found)"
fi
echo ""
echo "To deploy to your Ubuntu server:"
echo "1. Copy binary: scp port-checker-linux user@server:~/"
echo "2. SSH to server"
echo "3. Run: ./port-checker-linux"
echo ""
echo "The binary contains embedded .env and targets.txt"
echo "You can still override by creating .env or targets.txt on the server"

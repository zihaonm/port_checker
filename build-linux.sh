#!/bin/bash

set -e

# Parse arguments
BINARY_NAME="${1:-port-checker-linux}"
TARGETS_FILE="${2:-targets.txt}"

echo "Building $BINARY_NAME for Linux amd64..."
echo ""
echo "Note: This will embed .env and $TARGETS_FILE into the binary"
echo ""

# Check if files exist
if [ ! -f ".env" ]; then
    echo "WARNING: .env file not found - binary will not have embedded credentials"
fi

if [ ! -f "$TARGETS_FILE" ]; then
    echo "ERROR: Targets file '$TARGETS_FILE' not found"
    exit 1
fi

# If using a custom targets file, copy it to targets.txt for embedding
RESTORE_TARGETS=false
if [ "$TARGETS_FILE" != "targets.txt" ]; then
    if [ -f "targets.txt" ]; then
        echo "Backing up existing targets.txt..."
        cp targets.txt targets.txt.backup
        RESTORE_TARGETS=true
    fi
    echo "Copying $TARGETS_FILE to targets.txt for embedding..."
    cp "$TARGETS_FILE" targets.txt
fi

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Build for Linux amd64
echo "Cross-compiling for Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o "$BINARY_NAME"

# Restore original targets.txt if needed
if [ "$RESTORE_TARGETS" = true ]; then
    echo "Restoring original targets.txt..."
    mv targets.txt.backup targets.txt
elif [ "$TARGETS_FILE" != "targets.txt" ]; then
    echo "Cleaning up temporary targets.txt..."
    rm -f targets.txt
fi

echo ""
echo "✓ Build successful!"
echo "Binary created: ./$BINARY_NAME"
echo ""
echo "Embedded files:"
if [ -f ".env" ]; then
    echo "  ✓ .env"
else
    echo "  ✗ .env (not found)"
fi
echo "  ✓ $TARGETS_FILE"
echo ""
echo "To deploy to your Ubuntu server:"
echo "1. Copy binary: scp $BINARY_NAME user@server:~/"
echo "2. SSH to server"
echo "3. Run: ./$BINARY_NAME"
echo ""
echo "The binary contains embedded .env and $TARGETS_FILE"
echo "You can still override by creating .env or targets.txt on the server"

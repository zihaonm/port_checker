#!/bin/bash

set -e

echo "Building port-checker..."
echo ""
echo "Note: This will embed .env and targets.txt into the binary"
echo ""

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Build the binary
echo "Compiling binary..."
go build -o port-checker

echo ""
echo "✓ Build successful!"
echo "Binary created: ./port-checker"
echo ""
echo "To run: ./port-checker"

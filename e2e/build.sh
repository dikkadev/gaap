#!/bin/bash

# Exit on error
set -e

# Enable command logging
set -x

echo "=== Starting E2E Test Build Process ==="

# Get the directory containing this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
echo "Script directory: $SCRIPT_DIR"

echo "=== Building GAAP Binary ==="
echo "Changing to project root directory..."
cd "$SCRIPT_DIR/.."
pwd

echo "Building gaap binary..."
go build -v -o e2e/gaap ./cmd/gaap
echo "Binary built successfully"
ls -l e2e/gaap

echo "=== Building Test Container ==="
echo "Changing to e2e directory..."
cd "$SCRIPT_DIR"
pwd

echo "Building Docker image..."
DOCKER_BUILDKIT=1 docker build --progress=plain -t gaap-e2e-test:latest .
echo "Docker image built successfully"

echo "Verifying image..."
docker images | grep gaap-e2e-test

echo "=== Cleaning Up ==="
echo "Removing temporary binary..."
rm -fv gaap 

echo "=== Build Process Completed Successfully ==="

# Disable command logging
set +x
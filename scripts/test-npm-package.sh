#!/bin/bash

# Test script for npm package installation
set -e

echo "🧪 Testing sshifu npm package"
echo "=============================="
echo

# Create temp directory
TEST_DIR=$(mktemp -d)
echo "Test directory: $TEST_DIR"

# Copy package files
cp package.json scripts/install.js bin/*.js "$TEST_DIR/"
mkdir -p "$TEST_DIR/bin"
cp bin/*.js "$TEST_DIR/bin/"

cd "$TEST_DIR"

echo
echo "1. Testing install script..."
node scripts/install.js

echo
echo "2. Verifying binaries installed..."
for bin in sshifu sshifu-server sshifu-trust; do
  if [ -f "bin/$bin" ]; then
    echo "  ✓ bin/$bin exists"
    chmod +x "bin/$bin"
    "./bin/$bin" --help 2>&1 | head -1 || true
  else
    echo "  ✗ bin/$bin missing"
    exit 1
  fi
done

echo
echo "3. Testing wrapper scripts..."
for wrapper in sshifu.js sshifu-server.js sshifu-trust.js; do
  if [ -f "bin/$wrapper" ]; then
    echo "  ✓ bin/$wrapper exists"
  else
    echo "  ✗ bin/$wrapper missing"
    exit 1
  fi
done

echo
echo "✅ All tests passed!"
echo

# Cleanup
cd -
rm -rf "$TEST_DIR"

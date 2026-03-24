#!/bin/sh
set -e

echo "=== SSHifu Server Fly.io Entrypoint ==="

# Decode CA private key from environment variable
if [ -n "$CA_PRIVATE_KEY_B64" ]; then
    echo "Decoding CA private key..."
    echo "$CA_PRIVATE_KEY_B64" | base64 -d > /app/ca
    chmod 600 /app/ca
    echo "CA private key loaded successfully"
else
    echo "ERROR: CA_PRIVATE_KEY_B64 environment variable not set"
    exit 1
fi

# Copy CA public key if not already present
if [ ! -f /app/ca.pub ]; then
    echo "WARNING: ca.pub not found, generating from private key..."
    ssh-keygen -y -f /app/ca > /app/ca.pub
fi

echo "Starting sshifu-server..."
exec /app/sshifu-server

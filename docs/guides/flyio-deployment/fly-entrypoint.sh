#!/bin/sh
set -e

echo "=== SSHifu Server Fly.io Entrypoint ==="

# Check if config exists, if not run wizard
if [ ! -f /app/config.yml ]; then
    echo "No config found. Starting setup wizard..."
    echo "Follow the prompts to configure sshifu-server."
    echo ""
    /app/sshifu-server
    exit 0
fi

echo "Config found. Starting sshifu-server..."
exec /app/sshifu-server

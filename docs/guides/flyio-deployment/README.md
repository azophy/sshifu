# SSHifu Fly.io Deployment Assets

This folder contains all files needed to deploy sshifu-server on Fly.io.

## Files

| File | Description |
|------|-------------|
| `Dockerfile.fly` | Multi-stage Docker build for sshifu-server |
| `fly.toml.example` | Fly.io app configuration template |
| `config.fly.yml` | sshifu-server config with env var placeholders |
| `fly-entrypoint.sh` | Runtime script to decode CA key from secrets |

## Quick Setup

```bash
# Copy deployment files to project root
cp docs/guides/flyio-deployment/Dockerfile.fly .
cp docs/guides/flyio-deployment/fly.toml.example fly.toml
cp docs/guides/flyio-deployment/config.fly.yml config.fly.yml
mkdir -p scripts
cp docs/guides/flyio-deployment/fly-entrypoint.sh scripts/
chmod +x scripts/fly-entrypoint.sh

# Copy CA public key
cp ca.pub ca.pub
```

## Deployment Steps

1. Generate CA keys: `ssh-keygen -t ed25519 -f ca -N ""`
2. Encode CA key: `CA_PRIVATE_KEY_B64=$(base64 -w 0 ca)`
3. Create Fly.io app: `flyctl apps create sshifu-auth`
4. Set secrets (see full guide in `docs/guides/flyio-deployment.md`)
5. Deploy: `flyctl deploy`

## Full Documentation

See [flyio-deployment.md](../flyio-deployment.md) for the complete deployment tutorial.

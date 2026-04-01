# SSHifu Fly.io Deployment Assets

This folder contains all files needed to deploy sshifu-server on Fly.io using the interactive wizard.

## Files

| File | Description |
|------|-------------|
| `Dockerfile.fly` | Docker build using installer script |
| `fly.toml.example` | Fly.io app configuration template |
| `fly-entrypoint.sh` | Entrypoint that runs wizard on first run |
| `README.md` | This file |

## Quick Setup

```bash
# Copy deployment files to project root
cp docs/guides/flyio-deployment/Dockerfile.fly Dockerfile
cp docs/guides/flyio-deployment/fly.toml.example fly.toml
cp docs/guides/flyio-deployment/fly-entrypoint.sh fly-entrypoint.sh
chmod +x fly-entrypoint.sh
```

## Deployment Steps

1. Create Fly.io app: `flyctl apps create sshifu-auth`
2. Set OAuth secrets: `flyctl secrets set GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=... GITHUB_ALLOWED_ORG=...`
3. Deploy: `flyctl deploy`
4. Run wizard: Access via `flyctl ssh console` on first run
5. (Optional) Create volume to persist CA keys

## Key Features

- **Wizard-based setup**: Interactive configuration on first run
- **No manual CA key generation**: Wizard generates keys automatically
- **OAuth via secrets**: Credentials stored securely in Fly.io secrets
- **Optional persistence**: Use Fly.io volume to persist CA keys across restarts

## Full Documentation

See [docs/guides/flyio-deployment.md](../flyio-deployment.md) for the complete deployment tutorial.

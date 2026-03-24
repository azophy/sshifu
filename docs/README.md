# Sshifu Documentation

Welcome to the Sshifu documentation. Sshifu is a lightweight SSH authentication system using short-lived OpenSSH certificates with OAuth authentication.

## Documentation Overview

### For Users

| Document | Description |
|----------|-------------|
| [User Guide](guides/user-guide.md) | Getting started, installation, and usage |
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |

### For Administrators

| Document | Description |
|----------|-------------|
| [Server Setup Guide](guides/server-setup.md) | Deploying and configuring sshifu-server |
| [Fly.io Deployment](guides/flyio-deployment.md) | Deploy to Fly.io with managed secrets |
| [Configuration Reference](reference/configuration.md) | Complete configuration options |

### For Developers

| Document | Description |
|----------|-------------|
| [Development Guide](guides/development.md) | Architecture, building, and contributing |
| [API Reference](api/README.md) | HTTP API documentation |

---

## Quick Links

- **GitHub Repository:** [github.com/azophy/sshifu](https://github.com/azophy/sshifu)
- **Example Configuration:** [config.example.yml](../config.example.yml)
- **Product Requirements:** [PRD.md](../PRD.md)

---

## What is Sshifu?

Sshifu (SSH + 師傅 "master") provides:

- 🔐 **Short-lived SSH certificates** - Automatic issuance with configurable TTL
- 🌐 **GitHub OAuth authentication** - Authenticate via GitHub organization membership
- 🛠️ **Standard OpenSSH compatibility** - Works with existing `ssh` command
- 📦 **Minimal infrastructure** - Single server, no database required
- 👥 **Designed for small teams** - Optimized for teams with <50 users

---

## Architecture Overview

```
┌─────────────┐
│ User CLI    │  sshifu
│ (sshifu)    │
└──────┬──────┘
       │
       │ 1. Start login session
       │ 2. Poll for approval
       │ 3. Request certificate
       ▼
┌─────────────────────────┐
│ sshifu-server           │
│ - OAuth gateway         │
│ - SSH Certificate Auth  │
└─────────────────────────┘
       │
       │ Configure trust
       ▼
┌─────────────────────────┐
│ Target SSH Server       │
│ (configured via         │
│  sshifu-trust)          │
└─────────────────────────┘
```

---

## Getting Started

### For Users

1. Build or download the `sshifu` CLI
2. Run: `sshifu auth.example.com user@target-server`
3. Open the login URL in your browser
4. Authenticate via GitHub
5. SSH connection established!

See the [User Guide](guides/user-guide.md) for detailed instructions.

### For Administrators

1. Deploy `sshifu-server` with OAuth configuration
2. Run `sshifu-trust` on each target SSH server
3. Users can now authenticate and connect

See the [Server Setup Guide](guides/server-setup.md) for detailed instructions.

---

## Components

| Tool | Purpose |
|------|---------|
| `sshifu` | CLI for users to authenticate and connect |
| `sshifu-server` | OAuth gateway and Certificate Authority |
| `sshifu-trust` | Configures SSH servers to trust the CA |

---

## Authentication Flow

1. User runs `sshifu` CLI
2. CLI starts a login session and displays a URL
3. User opens URL and authenticates via GitHub OAuth
4. Server verifies GitHub organization membership
5. CLI obtains an SSH certificate
6. CLI connects via SSH with the certificate

See the [API Reference](api/README.md) for endpoint details.

---

## Configuration Example

```yaml
server:
  listen: ":8080"
  public_url: https://auth.example.com

ca:
  private_key: ./ca
  public_key: ./ca.pub

cert:
  ttl: 8h
  extensions:
    permit-pty: true
    permit-port-forwarding: true
    permit-agent-forwarding: true
    permit-x11-forwarding: true

auth:
  providers:
    - name: github
      type: github
      client_id: YOUR_GITHUB_CLIENT_ID
      client_secret: YOUR_GITHUB_CLIENT_SECRET
      allowed_org: your-github-org
```

See the [Configuration Reference](reference/configuration.md) for all options.

---

## Support

- **Issues:** [GitHub Issues](https://github.com/azophy/sshifu/issues)
- **Discussions:** [GitHub Discussions](https://github.com/azophy/sshifu/discussions)

---

## License

[MIT License](../LICENSE)

# Sshifu

<div align="center">
  <img src="logo.png" alt="Sshifu Logo" width="250px">
</div>

**Sshifu** (SSH + Fu / 師傅 "master") is a lightweight SSH authentication system that uses short-lived OpenSSH certificates with OAuth authentication (GitHub organizations).

A minimal alternative to complex SSH access platforms like Teleport, while remaining fully compatible with existing OpenSSH tooling.

## Features

- 🔐 **Short-lived SSH certificates** - Automatic certificate issuance with configurable TTL (default 8 hours)
- 🌐 **GitHub OAuth authentication** - Authenticate users via GitHub organization membership
- 🛠️ **Standard OpenSSH compatibility** - Works with existing `ssh` command without workflow changes
- 📦 **Minimal infrastructure** - Single server component, no database required
- 👥 **Designed for small teams** - Optimized for teams with <50 users

## Quick Start

### Prerequisites

- **Publicly accessible address** for the sshifu-server (for testing, you can use [localhost.run](https://localhost.run) to expose your local server)
- **GitHub OAuth Client ID & Secret** from your GitHub organization
- **SSH server** with the same username as the GitHub username you want to log in with

### Step 1: Run sshifu-server

```bash
npx sshifu-server
```

Follow the interactive prompts to configure:
- Server listen address (e.g., `:8080`)
- Public URL (your publicly accessible address)
- GitHub OAuth Client ID and Secret
- Allowed GitHub organization

### Step 2: Configure SSH trust

On your target SSH server, run:

```bash
sudo npx sshifu-trust <sshifu-server-public-url>
```

Example:
```bash
sudo npx sshifu-trust https://your-server.ngrok.io
```

This configures the SSH server to trust certificates issued by sshifu.

### Step 3: Connect from another machine

From a different machine, connect to your SSH server:

```bash
npx sshifu <sshifu-server-address> <username>@<target-ssh-server>
```

Example:
```bash
npx sshifu https://your-server.ngrok.io user@target-server.com
```

The first time you connect:
1. A login URL will be displayed
2. Open the URL in your browser and authenticate via GitHub
3. The CLI will automatically detect approval and obtain a certificate
4. SSH connection will be established

## Architecture

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

## Components

| Tool | Purpose |
|------|---------|
| `sshifu` | CLI used by users to authenticate and connect to SSH servers |
| `sshifu-server` | Web server acting as OAuth gateway and SSH Certificate Authority |
| `sshifu-trust` | Server-side CLI to configure SSH servers to trust the Sshifu CA |

## Installation Options

#### Option 1: Install Globally via npm

For frequent use, install globally:

```bash
npm install -g sshifu
```

Then use commands directly:

```bash
sshifu auth.example.com user@target-server.com
sshifu-server
sshifu-trust auth.example.com
```

#### Option 2: Pre-built Binary

Download the latest release for your platform from the [releases page](https://github.com/azophy/sshifu/releases):

```bash
# Linux (amd64)
curl -L https://github.com/azophy/sshifu/releases/latest/download/sshifu-linux-amd64.tar.gz | tar xz
sudo mv sshifu* /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/azophy/sshifu/releases/latest/download/sshifu-darwin-amd64.tar.gz | tar xz
sudo mv sshifu* /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/azophy/sshifu/releases/latest/download/sshifu-darwin-arm64.tar.gz | tar xz
sudo mv sshifu* /usr/local/bin/

# Windows (amd64)
curl -L https://github.com/azophy/sshifu/releases/latest/download/sshifu-windows-amd64.zip -o sshifu.zip
unzip sshifu.zip
# Move sshifu.exe to a directory in your PATH
```

#### Option 3: Build from Source

Requires Go 1.25+:

```bash
go build ./cmd/sshifu
go build ./cmd/sshifu-server
go build ./cmd/sshifu-trust
```

## Getting GitHub OAuth Client ID & Secret

To configure OAuth authentication with GitHub, you need to create a GitHub OAuth App:

1. Go to your GitHub organization's settings: `https://github.com/organizations/<your-org>/settings`
2. Navigate to **Developer settings** (left sidebar)
3. Click **OAuth Apps** → **New OAuth App**
4. Fill in the application details:
   - **Application name**: `sshifu` (or any descriptive name)
   - **Homepage URL**: Your sshifu-server public URL (e.g., `https://auth.example.com`)
   - **Authorization callback URL**: Same as your sshifu-server public URL (e.g., `https://auth.example.com`)
5. Click **Register application**
6. After registration, you'll see your **Client ID** - copy it
7. Click **Generate a new client secret** and copy the secret

> ⚠️ **Important**: The client secret is only shown once. Store it securely and never commit it to version control.

## Authentication Flow

```
┌──────────┐                    ┌───────────────┐                    ┌──────────┐
│  sshifu  │                    │ sshifu-server │                    │  GitHub  │
└────┬─────┘                    └───────┬───────┘                    └────┬─────┘
     │                                  │                                  │
     │  POST /api/v1/login/start        │                                  │
     │─────────────────────────────────>│                                  │
     │                                  │                                  │
     │  session_id, login_url           │                                  │
     │<─────────────────────────────────│                                  │
     │                                  │                                  │
     │  [Display login URL to user]     │                                  │
     │                                  │                                  │
     │                                  │  [User opens URL in browser]     │
     │                                  │                                  │
     │                                  │  Redirect to GitHub OAuth        │
     │                                  │─────────────────────────────────>│
     │                                  │                                  │
     │                                  │  User authenticates              │
     │                                  │<─────────────────────────────────│
     │                                  │                                  │
     │                                  │  Verify org membership           │
     │                                  │─────────────────────────────────>│
     │                                  │                                  │
     │  GET /api/v1/login/status        │  Session approved                │
     │─────────────────────────────────>│                                  │
     │                                  │                                  │
     │  status: approved, access_token  │                                  │
     │<─────────────────────────────────│                                  │
     │                                  │                                  │
     │  POST /api/v1/sign/user          │                                  │
     │─────────────────────────────────>│                                  │
     │                                  │                                  │
     │  SSH certificate                 │                                  │
     │<─────────────────────────────────│                                  │
     │                                  │                                  │
     │  ssh -o CertificateFile=<cert>   │                                  │
     │─────────────────────────────────>│                                  │
     │         [SSH connection established]                                  │
```

## Configuration

### Server Configuration

| Field | Description | Default |
|-------|-------------|---------|
| `server.listen` | Address to listen on | `:8080` |
| `server.public_url` | Public URL of the server | Required |
| `ca.private_key` | Path to CA private key | `./ca` |
| `ca.public_key` | Path to CA public key | `./ca.pub` |
| `cert.ttl` | Certificate time-to-live | `8h` |
| `auth.providers` | OAuth provider configurations | Required |

### Certificate Extensions

By default, issued certificates include:
- `permit-pty` - Allow pseudo-terminal allocation
- `permit-port-forwarding` - Allow TCP port forwarding
- `permit-agent-forwarding` - Allow SSH agent forwarding
- `permit-x11-forwarding` - Allow X11 forwarding

## Project Structure

```
sshifu/
├── cmd/
│   ├── sshifu/          # User CLI
│   ├── sshifu-server/   # Server component
│   └── sshifu-trust/    # Server setup tool
├── internal/
│   ├── api/             # HTTP API handlers
│   ├── cert/            # SSH certificate operations
│   ├── config/          # Configuration loading
│   ├── oauth/           # OAuth provider implementations
│   ├── session/         # Login session management
│   └── ssh/             # SSH utilities
├── web/                 # Web frontend (login pages)
├── config.example.yml   # Example configuration
└── go.mod
```

## Security Considerations

- **Short-lived certificates** reduce the impact of compromised keys
- **CA private key** should be stored securely on the server
- **GitHub organization membership** is verified on each login
- **No long-term secrets** stored on client machines
- **Transparent authorization** - the target server's OS determines final access permissions

## Limitations (v1)

The following features are intentionally out of scope for the initial release:

- Role-based access control (RBAC)
- Server access policies
- Automatic account provisioning on SSH servers
- Session recording or audit logging
- Admin dashboard
- Certificate revocation

## Requirements

- OpenSSH 6.7+ (for certificate support)
- Linux/Unix-like operating system (server components)
- Windows, macOS, or Linux (client CLI)

## License

[MIT License](LICENSE)

## Contributing

Contributions are welcome! Please read the contributing guidelines before submitting pull requests.

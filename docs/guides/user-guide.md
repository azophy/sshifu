# User Guide

This guide covers how to use Sshifu to authenticate and connect to SSH servers.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Authentication Flow](#authentication-flow)
- [Certificate Management](#certificate-management)
- [Common Use Cases](#common-use-cases)

---

## Overview

Sshifu is a CLI tool that authenticates you via OAuth (GitHub organizations) and obtains short-lived SSH certificates. Once authenticated, you can use standard SSH commands to connect to servers that trust your organization's Certificate Authority (CA).

**Key benefits:**
- No long-term SSH keys to manage
- Automatic certificate renewal
- Works with existing SSH workflows
- Certificates expire automatically (default: 8 hours)

---

## Prerequisites

Before using Sshifu, ensure you have:

1. **Go 1.25+** (for building from source)
2. **OpenSSH 6.7+** (for certificate support)
3. **GitHub account** with membership in an authorized organization
4. **Access to a sshifu-server** instance

---

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/azophy/sshifu.git
cd sshifu

# Build the CLI
go build -o sshifu ./cmd/sshifu

# Move to your PATH (optional)
mv sshifu /usr/local/bin/
```

### Option 2: Pre-built Binary

Download the latest release for your platform from the [releases page](https://github.com/azophy/sshifu/releases).

---

## Quick Start

### Step 1: Connect to a Server

```bash
sshifu auth.example.com user@target-server.com
```

Where:
- `auth.example.com` is your organization's sshifu-server URL
- `user@target-server.com` is the target SSH server

### Step 2: Authenticate via Browser

On first run (or when your certificate expires), you'll see:

```
Using identity key: /home/user/.ssh/id_ed25519
No valid certificate found, starting login flow...

Please open this URL in your browser to authenticate:
  https://auth.example.com/login/abc123

Waiting for authentication...
```

1. Open the URL in your browser
2. Sign in with GitHub
3. The CLI will automatically detect when authentication is complete

### Step 3: SSH Connection Established

Once authenticated, Sshifu:
1. Obtains an SSH certificate
2. Installs the CA public key to your `known_hosts`
3. Executes SSH with the certificate

```
Certificate saved to: /home/user/.ssh/id_ed25519-cert.pub
CA key added to known_hosts
Connecting via SSH...
```

---

## Authentication Flow

```
┌─────────────┐                    ┌───────────────┐
│   You       │                    │ sshifu-server │
└──────┬──────┘                    └───────┬───────┘
       │                                   │
       │  1. Start login session           │
       │──────────────────────────────────>│
       │                                   │
       │  2. Return login URL              │
       │<──────────────────────────────────│
       │                                   │
       │  3. Open URL in browser           │
       │     ┌───────────────────────┐     │
       │     │  GitHub OAuth         │     │
       │     │  - Sign in            │     │
       │     │  - Verify org member  │     │
       │     └───────────────────────┘     │
       │                                   │
       │  4. Poll for approval             │
       │──────────────────────────────────>│
       │                                   │
       │  5. Return access token           │
       │<──────────────────────────────────│
       │                                   │
       │  6. Request certificate           │
       │──────────────────────────────────>│
       │                                   │
       │  7. Return SSH certificate        │
       │<──────────────────────────────────│
       │                                   │
       │  8. Connect via SSH               │
       │                                   │
```

---

## Certificate Management

### Certificate Location

Certificates are stored alongside your SSH key:

```
~/.ssh/id_ed25519           # Private key
~/.ssh/id_ed25519.pub       # Public key
~/.ssh/id_ed25519-cert.pub  # Certificate (auto-generated)
```

### Certificate Validity

- **Default TTL:** 8 hours
- **Automatic reuse:** Valid certificates are reused until expiration
- **Auto-renewal:** Expired certificates trigger a new login

### Check Certificate Status

View certificate details:

```bash
ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub
```

Example output:

```
Type: ssh-ed25519-cert-v01@openssh.com
User Certificate:
        Serial: 1234567890
        Valid: from 2024-01-15T10:00:00 to 2024-01-15T18:00:00
        Principals: your-github-username
        Extensions: permit-pty, permit-port-forwarding, ...
```

### Force Re-authentication

To force a new login, delete the certificate:

```bash
rm ~/.ssh/id_ed25519-cert.pub
sshifu auth.example.com user@target-server.com
```

---

## Common Use Cases

### Using a Custom Identity Key

```bash
sshifu auth.example.com -i ~/.ssh/work_key user@target-server.com
```

### With SSH Arguments

```bash
# Port forwarding
sshifu auth.example.com -L 8080:localhost:80 user@target-server.com

# Agent forwarding
sshifu auth.example.com -A user@target-server.com

# Specific port
sshifu auth.example.com -p 2222 user@target-server.com
```

### Multiple Servers

Once authenticated, you can connect to any server that trusts your CA:

```bash
# First connection (triggers login)
sshifu auth.example.com server1.example.com

# Subsequent connections (reuse certificate)
sshifu auth.example.com server2.example.com
sshifu auth.example.com server3.example.com
```

### In Scripts

Sshifu works in automated scripts. The certificate is reused until expiration:

```bash
#!/bin/bash

# Connect and run commands
sshifu auth.example.com user@server.com "hostname && uptime"

# SCP files
sshifu auth.example.com user@server.com "cat /var/log/app.log" > app.log
```

---

## Next Steps

- [Configuration Reference](../reference/configuration.md) - Server configuration options
- [API Reference](../api/README.md) - API endpoints for integration
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

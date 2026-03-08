# Server Setup Guide

This guide covers deploying and configuring sshifu-server and setting up target SSH servers.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Deploying sshifu-server](#deploying-sshifu-server)
- [Configuring OAuth](#configuring-oauth)
- [Setting Up Target Servers](#setting-up-target-servers)
- [Running in Production](#running-in-production)

---

## Overview

The Sshifu system consists of three components:

| Component | Purpose | Runs On |
|-----------|---------|---------|
| `sshifu-server` | OAuth gateway + Certificate Authority | Central server |
| `sshifu-trust` | Configures SSH servers to trust CA | Target SSH servers |
| `sshifu` | User CLI for authentication | User workstations |

---

## Prerequisites

### For sshifu-server

- **Server:** Linux/Unix with network access
- **Go 1.25+** (for building)
- **Public URL** accessible by users and target servers
- **GitHub OAuth App** (see [Configuring OAuth](#configuring-oauth))

### For Target Servers

- **SSH access** (sudo required)
- **OpenSSH 6.7+**
- **systemd** or **init** system for service management

---

## Deploying sshifu-server

### Step 1: Build the Server

```bash
git clone https://github.com/azophy/sshifu.git
cd sshifu

go build -o sshifu-server ./cmd/sshifu-server
go build -o sshifu-trust ./cmd/sshifu-trust
go build -o sshifu ./cmd/sshifu
```

### Step 2: Run the Server

```bash
./sshifu-server
```

On first run, the setup wizard will guide you through configuration:

```
Configuration not found. Starting setup wizard...

Server public URL: https://auth.example.com
CA private key path [./ca]: 
CA public key path [./ca.pub]: 
GitHub Client ID: 
GitHub Client Secret: 
GitHub Organization: your-org
```

### Step 3: Verify Server is Running

```bash
curl https://auth.example.com/api/v1/ca/pub
```

Expected response:

```json
{
  "public_key": "ssh-ed25519 AAAA..."
}
```

---

## Configuring OAuth

### GitHub OAuth App Setup

1. Go to your GitHub organization settings:
   ```
   https://github.com/organizations/<your-org>/settings
   ```

2. Navigate to **Developer settings** → **OAuth Apps** → **New OAuth App**

3. Fill in the application details:

   | Field | Value |
   |-------|-------|
   | Application name | `sshifu` (or descriptive name) |
   | Homepage URL | `https://auth.example.com` |
   | Authorization callback URL | `https://auth.example.com/oauth/callback` |

4. Click **Register application**

5. Copy the **Client ID**

6. Click **Generate a new client secret** and copy it

> ⚠️ **Important:** The client secret is only shown once. Store it securely.

### Manual Configuration

Create `config.yml`:

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

### Running with Custom Config

```bash
./sshifu-server
```

The server automatically loads `config.yml` from the current directory.

---

## Setting Up Target Servers

### Step 1: Copy sshifu-trust to Target Server

```bash
scp sshifu-trust user@target-server:/tmp/
```

### Step 2: Run sshifu-trust

SSH into the target server:

```bash
ssh user@target-server
```

Run the trust setup (requires sudo):

```bash
sudo /tmp/sshifu-trust auth.example.com
```

This will:

1. Download the CA public key
2. Install it to `/etc/ssh/sshifu_ca.pub`
3. Request a host certificate
4. Install the host certificate
5. Update `sshd_config`
6. Restart the SSH daemon

### Step 3: Verify Setup

Check that the CA key was installed:

```bash
cat /etc/ssh/sshifu_ca.pub
```

Check sshd_config:

```bash
grep -E "TrustedUserCAKeys|HostCertificate" /etc/ssh/sshd_config
```

Expected output:

```
TrustedUserCAKeys /etc/ssh/sshifu_ca.pub
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub
```

Test SSH connection from a user machine:

```bash
sshifu auth.example.com user@target-server
```

---

## Running in Production

### Systemd Service

Create `/etc/systemd/system/sshifu-server.service`:

```ini
[Unit]
Description=Sshifu Server - SSH Certificate Authority
After=network.target

[Service]
Type=simple
User=sshifu
Group=sshifu
WorkingDirectory=/opt/sshifu
ExecStart=/opt/sshifu/sshifu-server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable sshifu-server
sudo systemctl start sshifu-server
sudo systemctl status sshifu-server
```

### Firewall Configuration

Allow incoming HTTPS traffic:

```bash
# ufw
sudo ufw allow 443/tcp

# firewalld
sudo firewall-cmd --add-port=443/tcp --permanent
sudo firewall-cmd --reload
```

### HTTPS/TLS Setup

For production, use a reverse proxy (nginx, Caddy, etc.) with TLS:

#### nginx Example

```nginx
server {
    listen 443 ssl;
    server_name auth.example.com;

    ssl_certificate /etc/letsencrypt/live/auth.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/auth.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Backup CA Keys

The CA private key is critical. Back it up securely:

```bash
# Backup
cp ca /secure/location/ca.backup
cp ca.pub /secure/location/ca.pub.backup

# Set restrictive permissions
chmod 600 /secure/location/ca.backup
```

### Monitoring

Monitor the server with:

```bash
# Check service status
sudo systemctl status sshifu-server

# View logs
sudo journalctl -u sshifu-server -f

# Check API health
curl https://auth.example.com/api/v1/ca/pub
```

---

## Next Steps

- [User Guide](guides/user-guide.md) - For end users
- [Configuration Reference](../reference/configuration.md) - All config options
- [Troubleshooting](troubleshooting.md) - Common issues

# Running sshifu-server as a systemd Service

This guide covers installing and managing sshifu-server as a systemd service on Linux systems using systemd (Ubuntu, Debian, CentOS, RHEL, Fedora, etc.).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Overview](#overview)
- [Step 1: Generate CA Keys](#step-1-generate-ca-keys)
- [Step 2: Install sshifu-server](#step-2-install-sshifu-server)
- [Step 3: Create Configuration](#step-3-create-configuration)
- [Step 4: Create systemd Service](#step-4-create-systemd-service)
- [Step 5: Enable and Start Service](#step-5-enable-and-start-service)
- [Step 6: Configure OAuth](#step-6-configure-oauth)
- [Step 7: Set Up HTTPS/TLS](#step-7-set-up-httptls)
- [Step 8: Set Up Target Servers](#step-8-set-up-target-servers)
- [Managing the Service](#managing-the-service)
- [Security Hardening](#security-hardening)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Linux system with systemd (Ubuntu 18.04+, Debian 9+, CentOS 7+, RHEL 7+, Fedora 28+)
- Root or sudo access
- A domain or subdomain for your sshifu-server (e.g., `auth.example.com`)
- GitHub OAuth App configured (see [Step 6: Configure OAuth](#step-6-configure-oauth))

---

## Overview

Running sshifu-server as a systemd service provides:

- **Automatic startup** on system boot
- **Process monitoring** and automatic restart on failure
- **Centralized logging** via journalctl
- **Resource isolation** with dedicated user/group
- **Easy management** with systemctl commands

```
┌─────────────────────────────────────────────────┐
│              Linux Server                       │
│                                                 │
│  ┌─────────────────────────────────────────┐   │
│  │         systemd (init system)           │   │
│  │                                         │   │
│  │  ┌─────────────────────────────────┐   │   │
│  │  │  sshifu-server.service          │   │   │
│  │  │  - Runs as sshifu user          │   │   │
│  │  │  - Auto-restart on failure      │   │   │
│  │  │  - Logs to journalctl           │   │   │
│  │  └─────────────────────────────────┘   │   │
│  └─────────────────────────────────────────┘   │
│                                                 │
│  Files:                                         │
│  - /opt/sshifu/sshifu-server (binary)          │
│  - /opt/sshifu/config.yml (config)             │
│  - /opt/sshifu/ca, ca.pub (CA keys)            │
│  - /etc/systemd/system/sshifu-server.service   │
└─────────────────────────────────────────────────┘
```

---

## Step 1: Generate CA Keys

Generate the SSH Certificate Authority keys:

```bash
# Create directory for CA keys
sudo mkdir -p /opt/sshifu
cd /opt/sshifu

# Generate CA private key (Ed25519)
ssh-keygen -t ed25519 -f ca -N "" -C "sshifu-ca"

# Verify the public key
cat ca.pub
```

**Important:** The CA private key (`ca`) is sensitive. It will be owned by the `sshifu` user with restricted permissions.

---

## Step 2: Install sshifu-server

### Option A: Download Pre-built Binary

```bash
# Download latest release (adjust architecture as needed)
cd /tmp
curl -LO https://github.com/azophy/sshifu/releases/latest/download/sshifu-server-linux-amd64

# Verify download (optional but recommended)
# curl -LO https://github.com/azophy/sshifu/releases/latest/download/sshifu-server-linux-amd64.sha256
# sha256sum -c sshifu-server-linux-amd64.sha256

# Install to /opt/sshifu
sudo mv sshifu-server-linux-amd64 /opt/sshifu/sshifu-server
sudo chmod +x /opt/sshifu/sshifu-server
```

### Option B: Build from Source

```bash
# Clone repository
git clone https://github.com/azophy/sshifu.git
cd sshifu

# Build
go build -o sshifu-server ./cmd/sshifu-server

# Install
sudo mkdir -p /opt/sshifu
sudo mv sshifu-server /opt/sshifu/
sudo chmod +x /opt/sshifu/sshifu-server
```

### Option C: Install via npm

```bash
npm install -g sshifu-server
```

Note the installation path (usually `/usr/lib/node_modules/sshifu-server/sshifu-server`).

---

## Step 3: Create Configuration

Create the configuration file at `/opt/sshifu/config.yml`:

```bash
sudo tee /opt/sshifu/config.yml << 'EOF'
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
EOF
```

Update the values:
- `public_url`: Your server's public URL
- `client_id`: GitHub OAuth Client ID
- `client_secret`: GitHub OAuth Client Secret
- `allowed_org`: Your GitHub organization name

### Set Permissions

```bash
# Create sshifu user and group
sudo useradd --system --no-create-home --shell /usr/sbin/nologin sshifu

# Set ownership
sudo chown -R sshifu:sshifu /opt/sshifu
sudo chmod 755 /opt/sshifu
sudo chmod 600 /opt/sshifu/ca
sudo chmod 644 /opt/sshifu/ca.pub
sudo chmod 600 /opt/sshifu/config.yml
```

---

## Step 4: Create systemd Service

Create the systemd service file at `/etc/systemd/system/sshifu-server.service`:

```bash
sudo tee /etc/systemd/system/sshifu-server.service << 'EOF'
[Unit]
Description=Sshifu Server - SSH Certificate Authority
Documentation=https://github.com/azophy/sshifu
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=sshifu
Group=sshifu
WorkingDirectory=/opt/sshifu
ExecStart=/opt/sshifu/sshifu-server
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ReadWritePaths=/opt/sshifu

# Resource limits
LimitNOFILE=65536
MemoryMax=512M
CPUQuota=50%

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sshifu-server

[Install]
WantedBy=multi-user.target
EOF
```

### Service File Explanation

| Directive | Purpose |
|-----------|---------|
| `User=sshifu` | Run as non-root user for security |
| `Restart=on-failure` | Auto-restart if crashed |
| `RestartSec=5s` | Wait 5 seconds before restart |
| `NoNewPrivileges=true` | Prevent privilege escalation |
| `ProtectSystem=strict` | Read-only filesystem except specified paths |
| `ReadWritePaths=/opt/sshifu` | Allow writes only to app directory |
| `MemoryMax=512M` | Limit memory usage |
| `CPUQuota=50%` | Limit CPU usage to 50% of one core |

---

## Step 5: Enable and Start Service

```bash
# Reload systemd to recognize new service
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable sshifu-server

# Start the service
sudo systemctl start sshifu-server

# Check status
sudo systemctl status sshifu-server
```

Expected output:

```
● sshifu-server.service - Sshifu Server - SSH Certificate Authority
     Loaded: loaded (/etc/systemd/system/sshifu-server.service; enabled)
     Active: active (running) since Tue 2026-03-24 10:00:00 UTC
   Main PID: 12345 (sshifu-server)
      Tasks: 8 (limit: 4915)
     Memory: 12.5M
        CPU: 150ms
     CGroup: /system.slice/sshifu-server.service
             └─12345 /opt/sshifu/sshifu-server
```

### Verify Server is Running

```bash
# Test local endpoint
curl http://localhost:8080/api/v1/ca/pub

# Expected response:
# {"public_key": "ssh-ed25519 AAAA..."}
```

---

## Step 6: Configure OAuth

### 6.1 Create GitHub OAuth App

1. Go to your GitHub organization settings:
   ```
   https://github.com/organizations/<your-org>/settings
   ```

2. Navigate to **Developer settings** → **OAuth Apps** → **New OAuth App**

3. Fill in the application details:

   | Field | Value |
   |-------|-------|
   | Application name | `sshifu` |
   | Homepage URL | `https://auth.example.com` |
   | Authorization callback URL | `https://auth.example.com/oauth/callback` |

4. Click **Register application**

5. Copy the **Client ID** and generate a **Client Secret**

### 6.2 Update Configuration

Edit `/opt/sshifu/config.yml`:

```bash
sudo nano /opt/sshifu/config.yml
```

Update the OAuth credentials:

```yaml
auth:
  providers:
    - name: github
      type: github
      client_id: "your-actual-client-id"
      client_secret: "your-actual-client-secret"
      allowed_org: "your-github-org"
```

### 6.3 Restart Service

```bash
sudo systemctl restart sshifu-server
```

---

## Step 7: Set Up HTTPS/TLS

For production, use a reverse proxy with TLS termination.

### Option A: nginx with Let's Encrypt

#### Install nginx and certbot

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install nginx certbot python3-certbot-nginx

# CentOS/RHEL
sudo yum install nginx certbot python3-certbot-nginx
```

#### Configure nginx

```bash
sudo tee /etc/nginx/sites-available/sshifu << 'EOF'
server {
    listen 80;
    server_name auth.example.com;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$server_name$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name auth.example.com;

    ssl_certificate /etc/letsencrypt/live/auth.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/auth.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_read_timeout 60s;
    }
}
EOF
```

#### Enable site and obtain certificate

```bash
# Create certbot directory
sudo mkdir -p /var/www/certbot

# Enable site
sudo ln -s /etc/nginx/sites-available/sshifu /etc/nginx/sites-enabled/

# Test nginx config
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx

# Obtain SSL certificate
sudo certbot --nginx -d auth.example.com
```

#### Update sshifu config

Update `/opt/sshifu/config.yml`:

```yaml
server:
  listen: ":8080"
  public_url: https://auth.example.com
```

Restart sshifu-server:

```bash
sudo systemctl restart sshifu-server
```

### Option B: Caddy (Automatic HTTPS)

Caddy automatically handles HTTPS with Let's Encrypt:

```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

Create Caddyfile:

```bash
sudo tee /etc/caddy/Caddyfile << 'EOF'
auth.example.com {
    reverse_proxy localhost:8080
}
EOF
```

Restart Caddy:

```bash
sudo systemctl restart caddy
```

---

## Step 8: Set Up Target Servers

On each target SSH server, run `sshifu-trust`:

```bash
# Download sshifu-trust
curl -LO https://github.com/azophy/sshifu/releases/latest/download/sshifu-trust-linux-amd64
chmod +x sshifu-trust-linux-amd64

# Run trust setup (requires sudo)
sudo ./sshifu-trust-linux-amd64 auth.example.com
```

### Verify Trust Setup

```bash
# Check CA key installation
cat /etc/ssh/sshifu_ca.pub

# Test connection from a user machine
sshifu auth.example.com user@target-server
```

---

## Managing the Service

### Check Status

```bash
# Service status
sudo systemctl status sshifu-server

# Detailed status
sudo systemctl status sshifu-server --no-pager -l
```

### View Logs

```bash
# Recent logs
sudo journalctl -u sshifu-server -n 50

# Follow logs in real-time
sudo journalctl -u sshifu-server -f

# Logs from today
sudo journalctl -u sshifu-server --since today

# Logs with specific priority
sudo journalctl -u sshifu-server -p err
```

### Start/Stop/Restart

```bash
# Start
sudo systemctl start sshifu-server

# Stop
sudo systemctl stop sshifu-server

# Restart
sudo systemctl restart sshifu-server

# Reload (if supported)
sudo systemctl reload sshifu-server
```

### Enable/Disable on Boot

```bash
# Enable (start on boot)
sudo systemctl enable sshifu-server

# Disable (don't start on boot)
sudo systemctl disable sshifu-server

# Check if enabled
sudo systemctl is-enabled sshifu-server
```

### Update Service

```bash
# Download new version
cd /tmp
curl -LO https://github.com/azophy/sshifu/releases/latest/download/sshifu-server-linux-amd64

# Stop service
sudo systemctl stop sshifu-server

# Replace binary
sudo mv sshifu-server-linux-amd64 /opt/sshifu/sshifu-server
sudo chmod +x /opt/sshifu/sshifu-server
sudo chown sshifu:sshifu /opt/sshifu/sshifu-server

# Start service
sudo systemctl start sshifu-server

# Verify
sudo systemctl status sshifu-server
```

---

## Security Hardening

### File Permissions

Ensure proper permissions:

```bash
# CA private key (most sensitive)
sudo chmod 600 /opt/sshifu/ca
sudo chown sshifu:sshifu /opt/sshifu/ca

# Config file (contains secrets)
sudo chmod 600 /opt/sshifu/config.yml
sudo chown sshifu:sshifu /opt/sshifu/config.yml

# Binary
sudo chmod 755 /opt/sshifu/sshifu-server
sudo chown sshifu:sshifu /opt/sshifu/sshifu-server
```

### Firewall Configuration

Allow only necessary ports:

```bash
# UFW (Ubuntu/Debian)
sudo ufw allow 443/tcp  # HTTPS
sudo ufw allow 22/tcp   # SSH (if needed)
sudo ufw enable

# firewalld (CentOS/RHEL)
sudo firewall-cmd --add-port=443/tcp --permanent
sudo firewall-cmd --add-port=22/tcp --permanent
sudo firewall-cmd --reload
```

### SELinux (if enabled)

```bash
# Check SELinux status
getenforce

# If enforcing, set context for sshifu-server
sudo semanage fcontext -a -t http_port_t "/opt/sshifu/sshifu-server"
sudo restorecon -v /opt/sshifu/sshifu-server
```

### Backup CA Keys

```bash
# Create backup directory
sudo mkdir -p /root/sshifu-backup

# Backup CA keys
sudo cp /opt/sshifu/ca /root/sshifu-backup/ca.$(date +%Y%m%d)
sudo cp /opt/sshifu/ca.pub /root/sshifu-backup/ca.pub.$(date +%Y%m%d)

# Set restrictive permissions
sudo chmod 600 /root/sshifu-backup/ca.*
```

---

## Troubleshooting

### Service Won't Start

Check status and logs:

```bash
sudo systemctl status sshifu-server
sudo journalctl -u sshifu-server -n 100 --no-pager
```

Common issues:

**Permission denied:**
```bash
# Check ownership
ls -la /opt/sshifu/

# Fix ownership
sudo chown -R sshifu:sshifu /opt/sshifu
```

**Port already in use:**
```bash
# Check what's using port 8080
sudo ss -tlnp | grep :8080

# Kill conflicting process or change port in config
```

**Config file error:**
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('/opt/sshifu/config.yml'))"
```

### OAuth Failures

Check configuration:

```bash
# Verify config
sudo cat /opt/sshifu/config.yml | grep -A 5 auth:

# Restart service
sudo systemctl restart sshifu-server

# Check logs
sudo journalctl -u sshifu-server -f
```

Verify GitHub OAuth app settings:
- Callback URL matches exactly: `https://auth.example.com/oauth/callback`
- Client ID and secret are correct
- Organization name is correct

### Certificate Errors

If users can't connect:

1. Verify CA keys match:
   ```bash
   sudo -u sshifu ssh-keygen -y -f /opt/sshifu/ca
   # Should match ca.pub content
   ```

2. Check target server trust:
   ```bash
   ssh user@target-server "cat /etc/ssh/sshifu_ca.pub"
   ```

3. Re-run sshifu-trust on target server if needed

### Performance Issues

Check resource usage:

```bash
# Memory and CPU
sudo systemctl status sshifu-server

# Detailed resource usage
sudo systemd-cgtop
```

Adjust resource limits in service file if needed:

```ini
[Service]
MemoryMax=1G
CPUQuota=100%
```

Reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart sshifu-server
```

### Connection Timeouts

Check network connectivity:

```bash
# Test local connection
curl http://localhost:8080/api/v1/ca/pub

# Test from external
curl https://auth.example.com/api/v1/ca/pub

# Check firewall
sudo ufw status
```

---

## Advanced Configuration

### Custom Working Directory

If you want to store files elsewhere:

```ini
[Service]
WorkingDirectory=/var/lib/sshifu
ExecStart=/opt/sshifu/sshifu-server
```

### Environment Variables

Pass environment variables:

```ini
[Service]
Environment="SSHIFU_LOG_LEVEL=debug"
Environment="SSHIFU_CONFIG=/etc/sshifu/config.yml"
```

### Multiple Instances

Run multiple instances on different ports:

```bash
# Create second service file
sudo cp /etc/systemd/system/sshifu-server.service /etc/systemd/system/sshifu-server@.service

# Modify ExecStart to use port from instance
# Then start: sudo systemctl start sshifu-server@8081
```

### Log Rotation

systemd journal handles log rotation automatically. To configure:

```bash
# Edit journald config
sudo nano /etc/systemd/journald.conf

# Set max size
SystemMaxUse=1G
```

---

## Next Steps

- [User Guide](user-guide.md) - For end users connecting to SSH servers
- [OAuth Providers](oauth-providers.md) - Configure additional OAuth providers
- [Fly.io Deployment](flyio-deployment.md) - Deploy to Fly.io instead
- [Troubleshooting](../troubleshooting.md) - Common issues and solutions

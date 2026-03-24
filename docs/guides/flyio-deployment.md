# Deploying sshifu-server on Fly.io

This guide covers deploying sshifu-server to [Fly.io](https://fly.io), a global edge cloud platform. We'll use Fly.io's secrets management to securely store the CA private key and OAuth credentials.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Overview](#overview)
- [Step 1: Generate CA Keys](#step-1-generate-ca-keys)
- [Step 2: Create Fly.io App](#step-2-create-flyio-app)
- [Step 3: Prepare Configuration](#step-3-prepare-configuration)
- [Step 4: Build Docker Image](#step-4-build-docker-image)
- [Step 5: Deploy to Fly.io](#step-5-deploy-to-flyio)
- [Step 6: Configure OAuth](#step-6-configure-oauth)
- [Step 7: Set Up Target Servers](#step-7-set-up-target-servers)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- [Fly.io CLI](https://fly.io/docs/flyctl/) installed and authenticated
- [Docker](https://docs.docker.com/get-docker/) installed locally
- A GitHub OAuth App configured (see [Configuring OAuth](#step-6-configure-oauth))
- A domain or subdomain for your sshifu-server (e.g., `auth.example.com`)

---

## Overview

The deployment architecture:

```
┌─────────────────────────────────────────────────────────┐
│                    Fly.io App                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │              sshifu-server container            │   │
│  │                                                 │   │
│  │  - config.yml (in image)                        │   │
│  │  - CA key (decoded from secret at runtime)      │   │
│  │  - OAuth secrets (from Fly.io secrets)          │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS
                          ▼
              Users & Target SSH Servers
```

**Security approach:**
- CA private key is stored as a base64-encoded Fly.io secret
- OAuth credentials are stored as Fly.io secrets
- Non-sensitive config.yml is baked into the Docker image
- Entrypoint script decodes secrets at runtime

---

## Step 1: Generate CA Keys

First, generate the SSH Certificate Authority keys:

```bash
# Generate CA private key (Ed25519)
ssh-keygen -t ed25519 -f ca -N "" -C "sshifu-ca"

# Verify the public key
cat ca.pub
```

**Important:** Store the private key (`ca`) securely. You'll need to encode it for Fly.io secrets.

---

## Step 2: Create Fly.io App

```bash
# Create a new Fly.io app
flyctl apps create sshifu-auth

# Or launch interactively
flyctl launch --no-deploy
```

Note your app name (e.g., `sshifu-auth`) and region.

---

## Step 3: Prepare Configuration

### 3.1 Encode CA Private Key

Encode the CA private key as base64:

```bash
# On Linux/macOS
CA_PRIVATE_KEY_B64=$(base64 -w 0 ca)
echo $CA_PRIVATE_KEY_B64

# On Windows (PowerShell)
$CA_PRIVATE_KEY_B64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes("ca"))
echo $CA_PRIVATE_KEY_B64
```

Save this value - you'll store it as a Fly.io secret.

### 3.2 Create config.yml

Copy the Fly.io config template from the docs folder:

```bash
cp docs/guides/flyio-deployment/config.fly.yml config.fly.yml
```

Or create a `config.fly.yml` manually:

```yaml
server:
  listen: ":8080"
  public_url: https://sshifu-auth.fly.dev

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
      client_id: ${GITHUB_CLIENT_ID}
      client_secret: ${GITHUB_CLIENT_SECRET}
      allowed_org: ${GITHUB_ALLOWED_ORG}
```

**Note:** OAuth credentials are loaded from environment variables (Fly.io secrets).

### 3.3 Create CA Public Key File

Copy the public key to `ca.pub` in your project directory:

```bash
cp ca.pub ca.pub
```

### 3.4 Create Entrypoint Script

Copy the entrypoint script from the docs folder:

```bash
cp docs/guides/flyio-deployment/fly-entrypoint.sh scripts/
chmod +x scripts/fly-entrypoint.sh
```

Or create `scripts/fly-entrypoint.sh` manually:

```bash
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
```

---

## Step 4: Build Docker Image

Copy the Dockerfile from the docs folder:

```bash
cp docs/guides/flyio-deployment/Dockerfile.fly .
```

Or create a `Dockerfile.fly` manually:

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sshifu-server ./cmd/sshifu-server

# Final stage
FROM alpine:3.19

# Install OpenSSH client (for ssh-keygen) and ca-certificates
RUN apk add --no-cache openssh-client ca-certificates bash

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/sshifu-server /app/sshifu-server

# Copy configuration files
COPY config.fly.yml /app/config.yml
COPY ca.pub /app/ca.pub

# Copy and set up entrypoint script
COPY scripts/fly-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create non-root user
RUN addgroup -g 1000 sshifu && \
    adduser -D -u 1000 -G sshifu sshifu

# Set ownership
RUN chown -R sshifu:sshifu /app

USER sshifu

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
```

---

## Step 5: Deploy to Fly.io

### 5.1 Create fly.toml

Copy the example fly.toml from the docs folder:

```bash
cp docs/guides/flyio-deployment/fly.toml.example fly.toml
```

Or create a `fly.toml` manually:

```toml
app = 'sshifu-auth'
primary_region = 'sjc'

[build]
  dockerfile = "Dockerfile"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = false
  auto_start_machines = true
  min_machines_running = 1
  processes = ['app']

  [http_service.concurrency]
    type = "connections"
    hard_limit = 250
    soft_limit = 200

[[vm]]
  memory = '512MB'
  cpu_kind = 'shared'
  cpus = 1
```

### 5.2 Set Fly.io Secrets

Store sensitive values as Fly.io secrets:

```bash
# CA private key (base64 encoded)
flyctl secrets set CA_PRIVATE_KEY_B64="$CA_PRIVATE_KEY_B64"

# GitHub OAuth credentials
flyctl Secrets set GITHUB_CLIENT_ID="your-client-id"
flyctl Secrets set GITHUB_CLIENT_SECRET="your-client-secret"
flyctl Secrets set GITHUB_ALLOWED_ORG="your-github-org"
```

### 5.3 Deploy

```bash
# Deploy the application
flyctl deploy

# Watch the deployment
flyctl logs
```

### 5.4 Verify Deployment

```bash
# Check app status
flyctl status

# Test the API endpoint
curl https://sshifu-auth.fly.dev/api/v1/ca/pub
```

Expected response:

```json
{
  "public_key": "ssh-ed25519 AAAA..."
}
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
   | Homepage URL | `https://sshifu-auth.fly.dev` |
   | Authorization callback URL | `https://sshifu-auth.fly.dev/oauth/callback` |

4. Click **Register application**

5. Copy the **Client ID** and generate a **Client Secret**

### 6.2 Update Secrets

If you haven't already, set the OAuth secrets:

```bash
flyctl secrets set \
  GITHUB_CLIENT_ID="your-client-id" \
  GITHUB_CLIENT_SECRET="your-client-secret" \
  GITHUB_ALLOWED_ORG="your-github-org"
```

### 6.3 Redeploy (if needed)

```bash
flyctl deploy
```

---

## Step 7: Set Up Target Servers

On each target SSH server, run `sshifu-trust` to establish trust:

```bash
# Download sshifu-trust binary
curl -LO https://github.com/azophy/sshifu/releases/latest/download/sshifu-trust-linux-amd64
chmod +x sshifu-trust-linux-amd64

# Run trust setup (requires sudo)
sudo ./sshifu-trust-linux-amd64 sshifu-auth.fly.dev
```

This will:
1. Download the CA public key
2. Configure the SSH server to trust it
3. Request and install a host certificate
4. Restart the SSH daemon

### Verify Trust Setup

```bash
# Check CA key installation
cat /etc/ssh/sshifu_ca.pub

# Test connection from a user machine
sshifu sshifu-auth.fly.dev user@target-server
```

---

## Custom Domain Setup

To use your own domain (e.g., `auth.example.com`):

### 1. Add Custom Domain to Fly.io

```bash
flyctl certs add auth.example.com
```

### 2. Update DNS

Add a CNAME record pointing to your Fly.io app:

```
auth.example.com. CNAME sshifu-auth.fly.dev.
```

### 3. Update config.fly.yml

```yaml
server:
  listen: ":8080"
  public_url: https://auth.example.com
```

### 4. Update GitHub OAuth App

Update the OAuth app URLs to use your custom domain:
- **Homepage URL**: `https://auth.example.com`
- **Callback URL**: `https://auth.example.com/oauth/callback`

### 5. Redeploy

```bash
flyctl deploy
```

---

## Troubleshooting

### App Won't Start

Check logs:

```bash
flyctl logs
```

Common issues:
- **Missing secrets:** Verify all required secrets are set
  ```bash
  flyctl secrets list
  ```
- **Invalid base64:** Ensure CA key is properly encoded
  ```bash
  echo "$CA_PRIVATE_KEY_B64" | base64 -d | head -1
  # Should show: -----BEGIN OPENSSH PRIVATE KEY-----
  ```

### Certificate Errors

If users can't connect:

1. Verify CA public key matches private key:
   ```bash
   # Decode and verify
   flyctl ssh console
   echo $CA_PRIVATE_KEY_B64 | base64 -d > /tmp/ca
   ssh-keygen -y -f /tmp/ca
   # Should match ca.pub content
   ```

2. Check target server trust:
   ```bash
   ssh user@target-server "cat /etc/ssh/sshifu_ca.pub"
   ```

### OAuth Failures

Check OAuth configuration:

1. Verify callback URL matches exactly in GitHub OAuth app
2. Ensure organization name is correct
3. Check user is a member of the allowed organization

Test OAuth flow:

```bash
curl -v https://sshifu-auth.fly.dev/api/v1/login/start
```

### Performance Issues

Scale up resources in `fly.toml`:

```toml
[[vm]]
  memory = '1GB'
  cpu_kind = 'shared'
  cpus = 2
```

Apply changes:

```bash
flyctl deploy
```

---

## Maintenance

### Update Application

```bash
# Pull latest changes and redeploy
git pull
flyctl deploy
```

### Rotate CA Keys

1. Generate new CA keys locally
2. Re-encode as base64
3. Update secret:
   ```bash
   flyctl secrets set CA_PRIVATE_KEY_B64="new-base64-key"
   ```
4. Redeploy
5. Re-run `sshifu-trust` on all target servers

### Backup CA Keys

Always keep a secure backup of your CA private key:

```bash
# Store in a secure location
cp ca /secure/backup/location/sshifu-ca-$(date +%Y%m%d)
chmod 600 /secure/backup/location/sshifu-ca-*
```

---

## Cost Estimate

Fly.io free tier includes:
- Up to 3 shared-cpu-1x VMs with 256MB RAM
- 3GB persistent volume storage
- 160GB outbound transfer

A minimal sshifu-server deployment:
- **1 VM** (shared-cpu-1x, 512MB): ~$2/month
- **Transfer**: Most usage within free tier

See [Fly.io Pricing](https://fly.io/docs/about/pricing/) for current rates.

---

## Next Steps

- [User Guide](user-guide.md) - For end users connecting to SSH servers
- [OAuth Providers](oauth-providers.md) - Configure additional OAuth providers
- [Troubleshooting](../troubleshooting.md) - Common issues and solutions

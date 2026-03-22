# Authentik + SSHifu Integration Guide

A comprehensive guide to integrating SSHifu with Authentik for OIDC-based SSH authentication.

---

## Overview

This guide walks you through:
1. Deploying Authentik with Docker Compose
2. Configuring Authentik as an OIDC provider
3. Setting up SSHifu to use Authentik for authentication
4. Connecting to SSH servers with OIDC-authenticated certificates

---

## Prerequisites

- Docker and Docker Compose installed
- A domain or local hostname for Authentik (e.g., `auth.example.com`)
- A domain or local hostname for SSHifu (e.g., `sshifu.example.com`)
- Basic understanding of OAuth2/OIDC concepts

---

## Part 1: Deploy Authentik

### 1.1 Create Project Directory

```bash
mkdir authentik && cd authentik
```

### 1.2 Generate Secrets

Generate secure secrets for Authentik:

```bash
# Generate PostgreSQL password
echo "PG_PASS=$(openssl rand -base64 36)" > .env

# Generate Authentik secret key
echo "AUTHENTIK_SECRET_KEY=$(openssl rand -base64 60)" >> .env

# Disable error reporting (optional)
echo "AUTHENTIK_ERROR_REPORTING__ENABLED=false" >> .env
```

### 1.3 Create Docker Compose File

Create `docker-compose.yml`:

```yaml
version: "3.8"

services:
  postgresql:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_PASSWORD: ${PG_PASS}
      POSTGRES_USER: authentik
      POSTGRES_DB: authentik
    volumes:
      - pg_data:/var/lib/postgresql/data

  redis:
    image: redis:alpine
    restart: unless-stopped

  authentik-server:
    image: ghcr.io/goauthentik/server:latest
    restart: unless-stopped
    command: server
    environment:
      AUTHENTIK_REDIS__HOST: redis
      AUTHENTIK_POSTGRESQL__HOST: postgresql
      AUTHENTIK_POSTGRESQL__USER: authentik
      AUTHENTIK_POSTGRESQL__NAME: authentik
      AUTHENTIK_POSTGRESQL__PASSWORD: ${PG_PASS}
      AUTHENTIK_SECRET_KEY: ${AUTHENTIK_SECRET_KEY}
    ports:
      - "9000:9000"   # HTTP
      - "9443:9443"   # HTTPS
    depends_on:
      - postgresql
      - redis

  authentik-worker:
    image: ghcr.io/goauthentik/server:latest
    restart: unless-stopped
    command: worker
    environment:
      AUTHENTIK_REDIS__HOST: redis
      AUTHENTIK_POSTGRESQL__HOST: postgresql
      AUTHENTIK_POSTGRESQL__USER: authentik
      AUTHENTIK_POSTGRESQL__NAME: authentik
      AUTHENTIK_POSTGRESQL__PASSWORD: ${PG_PASS}
      AUTHENTIK_SECRET_KEY: ${AUTHENTIK_SECRET_KEY}
    depends_on:
      - postgresql
      - redis

volumes:
  pg_data:
```

### 1.4 Start Authentik

```bash
docker compose up -d
```

### 1.5 Complete Initial Setup

1. Open the initial setup wizard:
   ```
   http://localhost:9000/if/flow/initial-setup/
   ```

2. Create your admin account and configure basic settings

3. Log in to the admin panel:
   ```
   http://localhost:9000/if/admin/
   ```

---

## Part 2: Configure Authentik OIDC Provider

### 2.1 Create OAuth2/OIDC Provider

1. In the Authentik admin panel, navigate to **Applications → Providers → Create**

2. Select **OAuth2/OpenID Connect Provider**

3. Fill in the configuration:

   | Field | Value |
   |-------|-------|
   | **Name** | `sshifu` |
   | **Authorization flow** | `default-provider-authorization-explicit-consent` |
   | **Client type** | `Confidential` |
   | **Redirect URIs/Origins** | `https://sshifu.example.com/oauth/callback` |
   | **Scopes** | `openid`, `profile`, `email` |

4. Save the provider

5. **Copy the generated credentials:**
   - Client ID (auto-generated)
   - Client Secret (auto-generated - shown only once!)

### 2.2 Create Application

1. Go to **Applications → Applications → Create**

2. Fill in:

   | Field | Value |
   |-------|-------|
   | **Name** | `SSHifu` |
   | **Slug** | `sshifu` |
   | **Provider** | Select `sshifu` (created above) |

3. Save the application

### 2.3 Note Your OIDC URLs

Your OIDC configuration URLs will be:

| Purpose | URL |
|---------|-----|
| **Issuer URL** | `https://auth.example.com/application/o/sshifu/` |
| **Discovery Endpoint** | `https://auth.example.com/application/o/sshifu/.well-known/openid-configuration` |
| **Authorization Endpoint** | `https://auth.example.com/application/o/sshifu/authorize/` |
| **Token Endpoint** | `https://auth.example.com/application/o/sshifu/token/` |
| **Userinfo Endpoint** | `https://auth.example.com/application/o/sshifu/userinfo/` |

> **Note:** Replace `auth.example.com` with your actual Authentik domain.

---

## Part 3: Configure SSHifu

### 3.1 Create Configuration File

Create a `config.yml` file for SSHifu:

```yaml
server:
  listen: ":8080"
  public_url: https://sshifu.example.com

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
    - name: authentik
      type: oidc
      issuer: https://auth.example.com/application/o/sshifu/
      client_id: YOUR_CLIENT_ID_FROM_AUTHENTIK
      client_secret: YOUR_CLIENT_SECRET_FROM_AUTHENTIK
      principal_oauth_field_name: preferred_username
```

### 3.2 Configuration Fields Explained

| Field | Description | Example |
|-------|-------------|---------|
| `server.listen` | Port SSHifu listens on | `:8080` |
| `server.public_url` | Public URL of SSHifu server | `https://sshifu.example.com` |
| `issuer` | Authentik OIDC issuer URL | `https://auth.example.com/application/o/sshifu/` |
| `client_id` | From Authentik provider setup | `abc123...` |
| `client_secret` | From Authentik provider setup | `xyz789...` |
| `principal_oauth_field_name` | OIDC claim for SSH username | `preferred_username`, `email`, or `sub` |

### 3.3 Username Claim Options

The `principal_oauth_field_name` determines which Authentik user attribute becomes the SSH username:

```yaml
# Use preferred username (default, recommended)
principal_oauth_field_name: preferred_username

# Use email address (without @domain)
principal_oauth_field_name: email

# Use internal user ID
principal_oauth_field_name: sub

# Use custom claim (if configured in Authentik)
principal_oauth_field_name: ssh_username
```

---

## Part 4: Run SSHifu Server

### 4.1 Install SSHifu

Using npx (no installation required):

```bash
npx sshifu-server
```

Or install globally:

```bash
npm install -g sshifu-server
sshifu-server
```

### 4.2 First Run

On first run, SSHifu will:
1. Load configuration from `config.yml`
2. Generate CA keys if they don't exist
3. Initialize the OIDC provider
4. Start the web server

Expected output:
```
🔐 Sshifu Server - SSH Certificate Authority and OAuth Gateway

Configuration loaded from config.yml
Server will listen on: :8080
Public URL: https://sshifu.example.com
CA private key: ./ca
CA public key: ./ca.pub
Certificate TTL: 8h
OAuth providers: 1 configured
  - authentik (oidc)
✓ CA loaded successfully
✓ Session store initialized
✓ OAuth provider initialized: authentik (oidc)
✓ Routes configured

🚀 Server starting...
```

---

## Part 5: Test the Integration

### 5.1 Test OIDC Discovery

Verify SSHifu can reach Authentik:

```bash
curl https://auth.example.com/application/o/sshifu/.well-known/openid-configuration | jq
```

Expected fields:
- `authorization_endpoint`
- `token_endpoint`
- `userinfo_endpoint`
- `issuer`

### 5.2 Test Login Flow

1. **Start the login process:**

   ```bash
   curl -X POST http://localhost:8080/api/v1/login/start
   ```

   Response:
   ```json
   {
     "session_id": "abc123...",
     "login_url": "https://sshifu.example.com/login/abc123..."
   }
   ```

2. **Open the login URL** in your browser

3. **Select Authentik** as the provider (if multiple configured)

4. **Authenticate** with your Authentik credentials

5. **Verify success** - you should see "Authentication Successful!"

### 5.3 Test SSH Connection

Once you have a certificate, test SSH:

```bash
ssh -o CertificateFile=./user-cert.pub user@target-server
```

---

## Part 6: Configure Target SSH Servers

To accept SSHifu-issued certificates, configure your target SSH servers:

### 6.1 Copy CA Public Key

Get the CA public key from SSHifu:

```bash
curl https://sshifu.example.com/api/v1/ca/pub | jq -r '.public_key'
```

### 6.2 Install on Target Server

On each target SSH server:

```bash
# Create trusted CA directory
sudo mkdir -p /etc/ssh/trusted-ca

# Save the CA public key
echo "ssh-ed25519-cert-v01@openssh.com AAAA..." | sudo tee /etc/ssh/trusted-ca/sshifu-ca.pub

# Configure SSH to trust the CA
echo "TrustedUserCAKeys /etc/ssh/trusted-ca/sshifu-ca.pub" | sudo tee -a /etc/ssh/sshd_config

# Restart SSH
sudo systemctl restart sshd
```

### 6.3 Verify Trust Configuration

```bash
# Check sshd_config
sudo sshd -T | grep trustedusercakeys

# Should show:
# trustedusercakeys /etc/ssh/trusted-ca/sshifu-ca.pub
```

---

## Troubleshooting

### OIDC Discovery Fails

**Error:** `Failed to initialize OIDC provider`

**Solutions:**
1. Verify issuer URL is correct (must end with `/`)
2. Test discovery endpoint manually:
   ```bash
   curl https://auth.example.com/application/o/sshifu/.well-known/openid-configuration
   ```
3. Check network connectivity between SSHifu and Authentik
4. Ensure Authentik is accessible from SSHifu's network

### Redirect URI Mismatch

**Error:** `redirect_uri_mismatch` from Authentik

**Solutions:**
1. Verify `server.public_url` in SSHifu config matches exactly
2. Check redirect URI in Authentik provider settings
3. Ensure no trailing slash differences
4. Use HTTPS in production (HTTP only for local dev)

### Username Claim Not Found

**Error:** `Failed to retrieve user information`

**Solutions:**
1. Try different `principal_oauth_field_name` values
2. Check available claims in Authentik userinfo response:
   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" \
     https://auth.example.com/application/o/sshifu/userinfo/ | jq
   ```
3. Default to `preferred_username` or `email`

### Certificate Not Accepted by SSH Server

**Error:** `Certificate invalid` or access denied

**Solutions:**
1. Verify CA public key is correctly installed on target server
2. Check SSH server logs: `/var/log/auth.log` or `/var/log/secure`
3. Ensure `TrustedUserCAKeys` is configured in `sshd_config`
4. Verify certificate hasn't expired (check TTL in config)

---

## Advanced Configuration

### Custom Claims in Authentik

To use custom claims for SSH usernames:

1. In Authentik, create a **Property Mapping**:
   - Go to **Customization → Property Mappings**
   - Create new **Scope Mapping**
   - Add custom logic to return desired username

2. Add the mapping to your provider's scopes

3. Reference the claim in SSHifu config:
   ```yaml
   principal_oauth_field_name: custom_ssh_username
   ```

### Multiple Providers

You can configure multiple authentication providers:

```yaml
auth:
  providers:
    - name: authentik
      type: oidc
      issuer: https://auth.example.com/application/o/sshifu/
      client_id: ...
      client_secret: ...

    - name: github
      type: github
      client_id: ...
      client_secret: ...
      allowed_org: my-org
```

Users will see both options at login.

### Restrict Access by Group

To restrict SSH access to specific Authentik groups:

1. Add groups to the OIDC scope in Authentik
2. Use a custom claim that includes group membership
3. Implement custom verification logic (requires code modification)

---

## Security Best Practices

### 1. Secure CA Keys

```bash
# Set restrictive permissions
chmod 600 ./ca
chmod 644 ./ca.pub

# Store on encrypted volume in production
# Back up securely
```

### 2. Use HTTPS

Always use HTTPS in production:
- Configure reverse proxy (nginx, traefik, etc.)
- Use valid TLS certificates
- Never expose SSHifu directly to the internet without TLS

### 3. Short Certificate TTL

For high-security environments:

```yaml
cert:
  ttl: 4h  # or even shorter
```

### 4. Minimal Extensions

Disable unnecessary certificate extensions:

```yaml
cert:
  extensions:
    permit-pty: true
    permit-port-forwarding: false
    permit-agent-forwarding: false
    permit-x11-forwarding: false
```

---

## Quick Reference

### URLs

| Service | URL Pattern |
|---------|-------------|
| Authentik Admin | `https://auth.example.com/if/admin/` |
| Authentik OIDC Discovery | `https://auth.example.com/application/o/sshifu/.well-known/openid-configuration` |
| SSHifu Login | `https://sshifu.example.com/login/{session_id}` |
| SSHifu CA Public Key | `https://sshifu.example.com/api/v1/ca/pub` |

### Common Commands

```bash
# Start SSHifu server
npx sshifu-server

# Get CA public key
curl https://sshifu.example.com/api/v1/ca/pub

# Start login flow
curl -X POST http://localhost:8080/api/v1/login/start

# Check login status
curl http://localhost:8080/api/v1/login/status/{session_id}
```

---

## Next Steps

- [OAuth Provider Configuration](oauth-providers.md) - Detailed OAuth/OIDC reference
- [Server Setup Guide](server-setup.md) - Production deployment
- [Configuration Reference](../reference/configuration.md) - All SSHifu options
- [User Guide](user-guide.md) - End-user instructions

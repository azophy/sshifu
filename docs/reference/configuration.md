# Configuration Reference

Complete reference for sshifu-server configuration options.

## Configuration File

The server loads configuration from `config.yml` in the current working directory by default.

## Quick Example

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

---

## Server Configuration

### `server.listen`

**Type:** string  
**Required:** No  
**Default:** `":8080"`

The address and port the server listens on.

**Examples:**

```yaml
# Listen on all interfaces, port 8080
listen: ":8080"

# Listen on specific interface
listen: "127.0.0.1:8080"

# Listen on IPv6
listen: "[::]:8080"
```

---

### `server.public_url`

**Type:** string  
**Required:** Yes  
**Default:** None

The public URL of the sshifu-server. This is used for:
- OAuth callback URLs
- Generating login URLs for CLI clients
- Redirect URIs

**Examples:**

```yaml
public_url: https://auth.example.com
public_url: https://sshifu.internal.company.com
public_url: http://localhost:8080  # Development only
```

> ⚠️ **Important:** Must be accessible by both users (for OAuth) and target servers (for certificate signing).

---

## CA Configuration

### `ca.private_key`

**Type:** string  
**Required:** No  
**Default:** `"./ca"`

Path to the CA private key file. If the file doesn't exist, it will be generated on first run.

**Examples:**

```yaml
private_key: ./ca
private_key: /etc/sshifu/ca_key
private_key: /secure/path/to/ca
```

**Security Notes:**
- Store this file securely with restricted permissions (0600)
- Back up this key - losing it invalidates all issued certificates
- Never commit to version control

---

### `ca.public_key`

**Type:** string  
**Required:** No  
**Default:** `"./ca.pub"`

Path to the CA public key file. Generated automatically if it doesn't exist.

**Examples:**

```yaml
public_key: ./ca.pub
public_key: /etc/sshifu/ca_key.pub
```

---

## Certificate Configuration

### `cert.ttl`

**Type:** string (duration)  
**Required:** No  
**Default:** `"8h"`

Time-to-live for issued user certificates. Supports Go duration format.

**Examples:**

```yaml
ttl: 8h      # 8 hours
ttl: 24h     # 24 hours
ttl: 1h30m   # 1 hour 30 minutes
ttl: 15m     # 15 minutes
ttl: 48h     # 48 hours
```

**Recommendations:**
- Short TTL (1-8h): High security, frequent re-authentication
- Medium TTL (8-24h): Balanced approach
- Long TTL (24-72h): Convenience-focused

---

### `cert.extensions`

**Type:** map[string]bool  
**Required:** No  
**Default:** All enabled

Controls which extensions are included in issued certificates.

**Available Extensions:**

| Extension | Default | Description |
|-----------|---------|-------------|
| `permit-pty` | `true` | Allow pseudo-terminal allocation |
| `permit-port-forwarding` | `true` | Allow TCP port forwarding (`-L`, `-R`) |
| `permit-agent-forwarding` | `true` | Allow SSH agent forwarding (`-A`) |
| `permit-x11-forwarding` | `true` | Allow X11 forwarding (`-X`) |

**Examples:**

```yaml
# All extensions enabled (default)
extensions:
  permit-pty: true
  permit-port-forwarding: true
  permit-agent-forwarding: true
  permit-x11-forwarding: true

# Minimal extensions
extensions:
  permit-pty: true
  permit-port-forwarding: false
  permit-agent-forwarding: false
  permit-x11-forwarding: false

# Custom configuration
extensions:
  permit-pty: true
  permit-port-forwarding: true
  permit-agent-forwarding: false  # Disable agent forwarding
  permit-x11-forwarding: false    # Disable X11
```

---

## Authentication Configuration

### `auth.providers`

**Type:** array  
**Required:** Yes (at least one)  
**Default:** None

List of OAuth provider configurations. At least one provider must be configured.

---

## GitHub Provider

**Type:** `github`

Authenticate users via GitHub organization membership.

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Provider identifier |
| `type` | string | Yes | Must be `"github"` |
| `client_id` | string | Yes | GitHub OAuth Client ID |
| `client_secret` | string | Yes | GitHub OAuth Client Secret |
| `allowed_org` | string | Yes | GitHub organization name |

### Example

```yaml
auth:
  providers:
    - name: github
      type: github
      client_id: Ov23liEXAMPLE123
      client_secret: abc123def456ghi789jkl012mno345pqr678
      allowed_org: my-company
```

### Setup Instructions

1. Go to GitHub organization settings
2. Navigate to **Developer settings** → **OAuth Apps**
3. Create new OAuth App with:
   - **Homepage URL:** Your `server.public_url`
   - **Callback URL:** `{server.public_url}/oauth/callback`
4. Copy Client ID and Client Secret to config

---

## OIDC Provider (Optional)

**Type:** `oidc`

Authenticate users via OpenID Connect provider.

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Provider identifier |
| `type` | string | Yes | Must be `"oidc"` |
| `issuer` | string | Yes | OIDC issuer URL |
| `client_id` | string | Yes | OIDC Client ID |
| `client_secret` | string | Yes | OIDC Client Secret |
| `principal_oauth_field_name` | string | No | Field to extract username (default: `preferred_username`) |

### Example

```yaml
auth:
  providers:
    - name: oidc
      type: oidc
      issuer: https://accounts.google.com
      client_id: YOUR_OIDC_CLIENT_ID
      client_secret: YOUR_OIDC_CLIENT_SECRET
      principal_oauth_field_name: email
```

### Supported Providers

- Google
- Okta
- Auth0
- Keycloak
- Any OIDC-compatible provider

---

## Multiple Providers

Configure multiple OAuth providers:

```yaml
auth:
  providers:
    - name: github
      type: github
      client_id: YOUR_GITHUB_CLIENT_ID
      client_secret: YOUR_GITHUB_CLIENT_SECRET
      allowed_org: company-a
    
    - name: github-personal
      type: github
      client_id: YOUR_OTHER_CLIENT_ID
      client_secret: YOUR_OTHER_CLIENT_SECRET
      allowed_org: company-b
```

> ⚠️ **Note:** Currently only the first provider is used. Multiple provider support is planned for future versions.

---

## Environment Variables

Configuration values can be overridden with environment variables:

| Variable | Config Field |
|----------|--------------|
| `SSHIFU_SERVER_LISTEN` | `server.listen` |
| `SSHIFU_SERVER_PUBLIC_URL` | `server.public_url` |
| `SSHIFU_CA_PRIVATE_KEY` | `ca.private_key` |
| `SSHIFU_CA_PUBLIC_KEY` | `ca.public_key` |
| `SSHIFU_CERT_TTL` | `cert.ttl` |

**Example:**

```bash
export SSHIFU_SERVER_LISTEN=":443"
export SSHIFU_SERVER_PUBLIC_URL="https://auth.example.com"
./sshifu-server
```

---

## Configuration Validation

On startup, the server validates:

1. **Required fields** are present
2. **URL format** is valid
3. **OAuth credentials** are non-empty
4. **CA keys** can be loaded (or generated)

Invalid configuration causes the server to exit with an error message.

---

## Default Values Summary

| Field | Default |
|-------|---------|
| `server.listen` | `:8080` |
| `server.public_url` | (required) |
| `ca.private_key` | `./ca` |
| `ca.public_key` | `./ca.pub` |
| `cert.ttl` | `8h` |
| `cert.extensions.permit-pty` | `true` |
| `cert.extensions.permit-port-forwarding` | `true` |
| `cert.extensions.permit-agent-forwarding` | `true` |
| `cert.extensions.permit-x11-forwarding` | `true` |

---

## Example Configurations

### Development

```yaml
server:
  listen: ":8080"
  public_url: http://localhost:8080

ca:
  private_key: ./dev_ca
  public_key: ./dev_ca.pub

cert:
  ttl: 24h
  extensions:
    permit-pty: true
    permit-port-forwarding: true
    permit-agent-forwarding: true
    permit-x11-forwarding: true

auth:
  providers:
    - name: github
      type: github
      client_id: dev_client_id
      client_secret: dev_client_secret
      allowed_org: my-org
```

### Production (Small Team)

```yaml
server:
  listen: ":8080"
  public_url: https://auth.company.com

ca:
  private_key: /etc/sshifu/ca
  public_key: /etc/sshifu/ca.pub

cert:
  ttl: 8h
  extensions:
    permit-pty: true
    permit-port-forwarding: true
    permit-agent-forwarding: true
    permit-x11-forwarding: false

auth:
  providers:
    - name: github
      type: github
      client_id: Ov23liEXAMPLE123
      client_secret: secure_secret_here
      allowed_org: company-name
```

### Production (High Security)

```yaml
server:
  listen: "127.0.0.1:8080"
  public_url: https://auth.company.com

ca:
  private_key: /secure/hsm/ca
  public_key: /etc/sshifu/ca.pub

cert:
  ttl: 4h
  extensions:
    permit-pty: true
    permit-port-forwarding: false
    permit-agent-forwarding: false
    permit-x11-forwarding: false

auth:
  providers:
    - name: github
      type: github
      client_id: Ov23liEXAMPLE123
      client_secret: secure_secret_here
      allowed_org: company-name
```

---

## Next Steps

- [Server Setup Guide](../guides/server-setup.md) - Deployment instructions
- [API Reference](../api/README.md) - API documentation
- [Troubleshooting](troubleshooting.md) - Common issues

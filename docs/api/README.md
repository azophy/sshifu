# API Reference

This document describes the sshifu-server HTTP API endpoints.

## Base URL

All API endpoints are relative to your sshifu-server base URL:

```
https://auth.example.com/api/v1/
```

## Authentication

Most endpoints require authentication via OAuth access token, obtained through the login flow.

---

## Login Endpoints

### POST /api/v1/login/start

Initiates a new login session.

**Request:**

```http
POST /api/v1/login/start
Content-Type: application/json
```

**Response:**

```json
{
  "session_id": "abc123xyz",
  "login_url": "https://auth.example.com/login/abc123xyz"
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Unique session identifier |
| `login_url` | string | URL to open in browser for OAuth |

**Example (curl):**

```bash
curl -X POST https://auth.example.com/api/v1/login/start
```

---

### GET /api/v1/login/status/{session_id}

Polls the status of a login session.

**Request:**

```http
GET /api/v1/login/status/{session_id}
```

**Response (pending):**

```json
{
  "status": "pending"
}
```

**Response (approved):**

```json
{
  "status": "approved",
  "access_token": "gho_abc123xyz..."
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Session status: `pending`, `approved`, or `expired` |
| `access_token` | string | OAuth access token (only when approved) |

**Status Values:**

| Status | Description |
|--------|-------------|
| `pending` | User has not completed OAuth yet |
| `approved` | User authenticated successfully |
| `expired` | Session timed out (not returned, session is deleted) |

**Example (curl):**

```bash
curl https://auth.example.com/api/v1/login/status/abc123xyz
```

---

## Certificate Authority Endpoints

### GET /api/v1/ca/pub

Retrieves the CA public key.

**Request:**

```http
GET /api/v1/ca/pub
```

**Response:**

```json
{
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI..."
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `public_key` | string | OpenSSH format CA public key |

**Example (curl):**

```bash
curl https://auth.example.com/api/v1/ca/pub
```

**Usage:**

The CA public key is installed on target SSH servers to trust user certificates signed by this CA.

---

## Certificate Signing Endpoints

### POST /api/v1/sign/user

Signs a user SSH certificate.

**Request:**

```http
POST /api/v1/sign/user
Content-Type: application/json
```

**Request Body:**

```json
{
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
  "access_token": "gho_abc123xyz..."
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `public_key` | string | User's SSH public key in OpenSSH format |
| `access_token` | string | OAuth access token from login status |

**Response:**

```json
{
  "certificate": "ssh-ed25519-cert-v01@openssh.com AAAAIHNz..."
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `certificate` | string | Signed SSH certificate in OpenSSH format |

**Certificate Properties:**

| Property | Value |
|----------|-------|
| Type | User certificate |
| Principal | GitHub username |
| TTL | Configured (default: 8 hours) |
| Extensions | permit-pty, permit-port-forwarding, permit-agent-forwarding, permit-x11-forwarding |

**Example (curl):**

```bash
curl -X POST https://auth.example.com/api/v1/sign/user \
  -H "Content-Type: application/json" \
  -d '{
    "public_key": "ssh-ed25519 AAAA...",
    "access_token": "gho_abc123..."
  }'
```

**Error Responses:**

| Status Code | Description |
|-------------|-------------|
| `400 Bad Request` | Invalid public key or missing fields |
| `401 Unauthorized` | Invalid or expired access token |
| `500 Internal Server Error` | Certificate signing failed |

---

### POST /api/v1/sign/host

Signs a host SSH certificate.

**Request:**

```http
POST /api/v1/sign/host
Content-Type: application/json
```

**Request Body:**

```json
{
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
  "principals": ["hostname", "hostname.local", "192.168.1.100"],
  "ttl": "720h"
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `public_key` | string | Host's SSH public key in OpenSSH format |
| `principals` | array | Hostnames and IPs for the certificate |
| `ttl` | string | Certificate validity duration (optional, default: 720h) |

**Response:**

```json
{
  "certificate": "ssh-ed25519-cert-v01@openssh.com AAAAIHNz..."
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `certificate` | string | Signed host certificate in OpenSSH format |

**Certificate Properties:**

| Property | Value |
|----------|-------|
| Type | Host certificate |
| Principals | As provided in request |
| TTL | As provided (default: 720h / 30 days) |

**Example (curl):**

```bash
curl -X POST https://auth.example.com/api/v1/sign/host \
  -H "Content-Type: application/json" \
  -d '{
    "public_key": "ssh-ed25519 AAAA...",
    "principals": ["server1.example.com", "192.168.1.10"]
  }'
```

**Notes:**

- No authentication required (intentionally open for ease of setup)
- Used by `sshifu-trust` tool during server configuration

---

## OAuth Endpoints

These endpoints are used internally by the OAuth flow and are not part of the public API.

### GET /oauth/github/{state}

Initiates GitHub OAuth authentication.

**Request:**

```http
GET /oauth/github/{state}
```

Redirects to GitHub OAuth with the appropriate scopes.

---

### GET /oauth/callback

GitHub OAuth callback endpoint.

**Request:**

```http
GET /oauth/callback?code={code}&state={state}
```

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `code` | OAuth authorization code from GitHub |
| `state` | Session state identifier |

**Flow:**

1. Exchanges code for access token
2. Fetches user info from GitHub API
3. Verifies organization membership
4. Approves the login session
5. Redirects to success page

---

## Error Handling

### Error Response Format

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_request` | 400 | Missing or invalid parameters |
| `unauthorized` | 401 | Invalid or missing authentication |
| `forbidden` | 403 | Access denied |
| `not_found` | 404 | Resource not found |
| `internal_error` | 500 | Server error |

### Example Error Response

```json
{
  "error": "unauthorized",
  "message": "Invalid or expired access token"
}
```

---

## Rate Limiting

Currently, no rate limiting is implemented. For production deployments, consider:

- Using a reverse proxy with rate limiting (nginx, traefik)
- Implementing custom middleware
- Using API gateway solutions

---

## Versioning

All API endpoints are versioned under `/api/v1/`. Breaking changes will increment the major version.

---

## Complete Flow Example

```bash
# 1. Start login session
SESSION=$(curl -s -X POST https://auth.example.com/api/v1/login/start)
SESSION_ID=$(echo $SESSION | jq -r '.session_id')
LOGIN_URL=$(echo $SESSION | jq -r '.login_url')

echo "Open this URL: $LOGIN_URL"

# 2. Poll for approval (in a real implementation, poll every 2 seconds)
while true; do
  STATUS=$(curl -s "https://auth.example.com/api/v1/login/status/$SESSION_ID")
  STATUS_VALUE=$(echo $STATUS | jq -r '.status')
  
  if [ "$STATUS_VALUE" = "approved" ]; then
    ACCESS_TOKEN=$(echo $STATUS | jq -r '.access_token')
    echo "Authenticated!"
    break
  fi
  
  sleep 2
done

# 3. Generate SSH key (if not exists)
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N ""

# 4. Read public key
PUB_KEY=$(cat ~/.ssh/id_ed25519.pub)

# 5. Request certificate
CERT_RESPONSE=$(curl -s -X POST https://auth.example.com/api/v1/sign/user \
  -H "Content-Type: application/json" \
  -d "{\"public_key\": \"$PUB_KEY\", \"access_token\": \"$ACCESS_TOKEN\"}")

CERTIFICATE=$(echo $CERT_RESPONSE | jq -r '.certificate')

# 6. Save certificate
echo "$CERTIFICATE" > ~/.ssh/id_ed25519-cert.pub

# 7. Connect via SSH
ssh -o CertificateFile=~/.ssh/id_ed25519-cert.pub user@target-server
```

---

## Next Steps

- [User Guide](../guides/user-guide.md) - End user documentation
- [Configuration Reference](../reference/configuration.md) - Server configuration
- [Development Guide](../guides/development.md) - Building and contributing

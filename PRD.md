# PRD — Sshifu (Updated)

## 1. Executive Summary

Sshifu is a lightweight system that simplifies SSH authentication using **short-lived OpenSSH certificates** issued by a centralized Certificate Authority (CA) after users authenticate via OAuth providers (initially GitHub organizations).

Sshifu provides a minimal alternative to complex SSH access platforms (e.g., Teleport) while remaining fully compatible with existing OpenSSH tooling.

The system consists of three Go-based tools:

| Tool            | Purpose                                                          |
| --------------- | ---------------------------------------------------------------- |
| `sshifu`        | CLI used by users to authenticate and connect to SSH servers     |
| `sshifu-server` | Web server acting as OAuth gateway and SSH Certificate Authority |
| `sshifu-trust`  | Server-side CLI to configure SSH servers to trust the Sshifu CA  |

Authentication uses a **simplified login session flow**:

1. CLI starts a login session
2. User authenticates via browser
3. CLI polls server for completion
4. Server issues short-lived SSH certificate

The system is designed for **small teams (<50 users)** and emphasizes **simplicity, minimal infrastructure, and compatibility with existing SSH workflows**.

---

# 2. Goals

### Primary Goals

* Simplify SSH login using **short-lived SSH certificates**
* Authenticate users via **GitHub OAuth**
* Require minimal infrastructure
* Maintain compatibility with standard OpenSSH tools
* Avoid modifying user SSH workflows

### Secondary Goals

* Easy deployment
* Minimal server state
* Simple CLI experience

---

# 3. Non-Goals (V1)

The following features are intentionally **out of scope** for the initial release:

* Role-based access control (RBAC)
* Limiting which servers a user may access
* Automatic account provisioning on SSH servers
* Server registry or discovery
* Session recording
* Audit logging
* Admin dashboard
* Certificate TTL overrides from CLI

Authorization is intentionally **transparent**:
Any authenticated user may attempt SSH access; the **target server OS determines final access permissions**.

---

# 4. System Architecture

```
User CLI (sshifu)
        │
        │ start login session
        ▼
sshifu-server
        │
        │ OAuth login
        ▼
GitHub OAuth
```

The server performs two main roles:

1. OAuth authentication gateway
2. SSH Certificate Authority

---

# 5. Authentication Flow (Simplified Login Session)

Instead of implementing full OAuth device flow, Sshifu uses a **login session polling model**.

### Step 1 — CLI starts login session

```
POST /api/v1/login/start
```

Server response:

```json
{
  "session_id": "abc123",
  "login_url": "https://auth.example.com/login/abc123"
}
```

---

### Step 2 — CLI displays login URL

```
Please open this URL:

https://auth.example.com/login/abc123
```

---

### Step 3 — User authenticates in browser

Browser visits:

```
/login/{session_id}
```

Server redirects to GitHub OAuth.

After successful login:

* server retrieves GitHub username
* verifies GitHub organization membership
* session status becomes `approved`

---

### Step 4 — CLI polls login status

CLI periodically calls:

```
GET /api/v1/login/status/{session_id}
```

Response:

```json
{
  "status": "approved",
  "access_token": "xyz"
}
```

---

### Step 5 — CLI requests SSH certificate

```
POST /api/v1/sign/user
```

Server returns signed SSH certificate.

---

# 6. Certificate Issuance

Sshifu-server acts as the SSH CA.

A **single CA keypair** signs both:

| Certificate      | Purpose                      |
| ---------------- | ---------------------------- |
| User certificate | Client authentication        |
| Host certificate | Server identity verification |

---

## User Certificate

Properties:

| Field     | Value            |
| --------- | ---------------- |
| Type      | User certificate |
| Principal | GitHub username  |
| TTL       | default 8 hours  |
| Key       | user public key  |

Extensions (configurable):

```
permit-pty
permit-port-forwarding
permit-agent-forwarding
permit-x11-forwarding
```

---

## Host Certificate

Used by SSH servers configured with `sshifu-trust`.

Properties:

| Field | Value            |
| ----- | ---------------- |
| Type  | Host certificate |
| Key   | SSH host key     |
| TTL   | configurable     |

---

# 7. CLI Tool: `sshifu`

Primary tool used by users.

---

## Command Syntax

```
sshifu <sshifu-server> [ssh arguments]
```

Example:

```
sshifu auth.example.com target-server.com
```

Example with custom identity:

```
sshifu auth.example.com -i ~/.ssh/my_key target-server.com
```

sshifu executes:

```
ssh -o CertificateFile=<cert> [ssh args]
```

---

## CLI Workflow

1. Determine identity key (`-i` or default key)
2. Check if certificate exists
3. Verify certificate validity
4. If expired or missing:

   * perform login session
   * obtain certificate
5. Download CA public key if not installed
6. Append CA entry to:

```
~/.ssh/known_hosts
```

7. Execute system `ssh`

---

## Certificate Storage

Certificates stored alongside the user's key:

```
~/.ssh/id_ed25519-cert.pub
```

Certificates are reused until expiration.

---

# 8. Server Setup Tool: `sshifu-trust`

Used on SSH servers.

Command:

```
sudo sshifu-trust <sshifu-server>
```

Example:

```
sudo sshifu-trust auth.example.com
```

---

## Workflow

1. Download CA public key

```
GET /api/v1/ca/pub
```

2. Install CA file

```
/etc/ssh/sshifu_ca.pub
```

3. Retrieve SSH host key

```
/etc/ssh/ssh_host_ed25519_key.pub
```

4. Request host certificate

```
POST /api/v1/sign/host
```

5. Install host certificate

```
/etc/ssh/ssh_host_ed25519_key-cert.pub
```

6. Modify `sshd_config`

```
TrustedUserCAKeys /etc/ssh/sshifu_ca.pub
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub
```

7. Restart SSH daemon

```
systemctl restart sshd
```

---

# 9. Server Component: `sshifu-server`

The server performs:

| Responsibility             |
| -------------------------- |
| OAuth authentication       |
| Login session management   |
| SSH certificate signing    |
| CA public key distribution |

The server is stateless except for temporary login sessions.

---

# 10. Server Configuration

Configuration stored in:

```
config.yml
```

Default location: current working directory.

---

### Example Config

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
      client_id: ""
      client_secret: ""
      allowed_org: example-org

    - name: oidc
      type: oidc
      issuer: https://example.com
      client_id: ""
      client_secret: ""
      principal_oauth_field_name: preferred_username
```

---

# 11. Login Session Storage

Login sessions stored **in memory**.

Structure:

```
map[string]*LoginSession
```

Example struct:

```go
type LoginSession struct {
    ID string
    Status string
    Username string
    AccessToken string
    CreatedAt time.Time
}
```

Session states:

```
pending
approved
expired
```

Sessions expire automatically.

---

# 12. API Specification

All endpoints are versioned.

```
/api/v1/
```

---

## Start Login Session

```
POST /api/v1/login/start
```

Response:

```
session_id
login_url
```

---

## Login Status

```
GET /api/v1/login/status/{session_id}
```

Response:

```
pending
approved
```

---

## Get CA Public Key

```
GET /api/v1/ca/pub
```

---

## Sign User Certificate

```
POST /api/v1/sign/user
```

Input:

```
public_key
access_token
```

---

## Sign Host Certificate

```
POST /api/v1/sign/host
```

Input:

```
host_public_key
```

No authentication required.

---

# 13. Data Storage

Persistent storage:

| Item    | Storage    |
| ------- | ---------- |
| Config  | YAML       |
| CA keys | filesystem |

In-memory:

| Item           |
| -------------- |
| login sessions |

No database required for v1.

---

# 14. Go Project Structure

```
cmd/
  sshifu/
  sshifu-server/
  sshifu-trust/

internal/
  config/
  cert/
  auth/
  session/
  api/
  ssh/

web/
  login.html
```

HTML and JavaScript are embedded in the Go binary.

---

# 15. Performance Expectations

Expected usage:

| Metric     | Value |
| ---------- | ----- |
| Users      | ≤ 50  |
| Logins/day | ≤ 150 |
| Servers    | ≤ 50  |

The server is designed to operate comfortably within these limits.

---

# 16. Acceptance Criteria

### sshifu CLI

* performs login session flow
* obtains SSH certificate
* reuses valid certificates
* launches system ssh

### sshifu-trust

* installs CA trust
* retrieves host certificate
* configures sshd correctly

### sshifu-server

* supports GitHub OAuth login
* issues valid SSH certificates
* manages login sessions

---

# 17. Future Roadmap

Potential future improvements:

* server registry
* RBAC
* audit logs
* admin dashboard
* CLI `sshifu login` command
* certificate revocation
* server access policies
* QR-code login support


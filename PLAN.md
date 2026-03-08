Below is the **updated rough implementation plan** aligned with the **simplified login-session PRD**.
The goal is to **reach a working end-to-end SSH login as early as possible**, then iterate.

I structured this as **milestones with concrete tasks**, roughly in the order you should implement them.

---

# Milestone 1 — Project Bootstrap

Goal: repository builds and runs.

### 1. Initialize repository

```bash
go mod init github.com/yourorg/sshifu
```

---

### 2. Create project structure

```
sshifu/
├── cmd/
│   ├── sshifu/
│   │   └── main.go
│   ├── sshifu-server/
│   │   └── main.go
│   └── sshifu-trust/
│       └── main.go
│
├── internal/
│   ├── config/
│   ├── cert/
│   ├── session/
│   ├── oauth/
│   ├── api/
│   └── ssh/
│
├── web/
│   └── login.html
│
└── config.example.yml
```

---

### 3. Install dependencies

```bash
go get github.com/goccy/go-yaml
go get github.com/gorilla/sessions
go get golang.org/x/crypto/ssh
go get golang.org/x/oauth2
```

---

# Milestone 2 — Config + Setup Wizard

Goal: `sshifu-server` can start with configuration.

---

## 2.1 Implement config loader

File:

```
internal/config/config.go
```

Tasks:

* load YAML config
* validate fields
* apply defaults

Example struct:

```go
type Config struct {
  Server struct {
    Listen    string
    PublicURL string
  }

  CA struct {
    PrivateKey string
    PublicKey  string
  }

  Cert struct {
    TTL string
    Extensions map[string]bool
  }

  Auth struct {
    Providers []Provider
  }
}
```

---

## 2.2 Setup wizard

Behavior:

If `config.yml` not found:

```
sshifu-server
```

Launch interactive wizard.

Wizard asks:

```
Server public URL
CA key path
GitHub client id
GitHub client secret
GitHub org
```

---

## 2.3 Generate CA keys

Use Go SSH library.

Files generated:

```
ca
ca.pub
```

---

# Milestone 3 — SSH Certificate Authority

Goal: `sshifu-server` can sign SSH keys.

---

## 3.1 Load CA signer

File:

```
internal/cert/ca.go
```

Function:

```go
func LoadCA(privateKeyPath string) (ssh.Signer, error)
```

---

## 3.2 Implement user certificate signing

```
internal/cert/user_cert.go
```

Function:

```go
func SignUserKey(
  userKey ssh.PublicKey,
  principal string,
  ttl time.Duration,
) ([]byte, error)
```

Fields:

```
Type: ssh.UserCert
Principal: username
ValidBefore
ValidAfter
Extensions
```

---

## 3.3 Implement host certificate signing

```
internal/cert/host_cert.go
```

Function:

```
SignHostKey()
```

Certificate type:

```
ssh.HostCert
```

---

# Milestone 4 — Login Session System

Goal: implement simplified authentication session.

---

## 4.1 Session manager

```
internal/session/session.go
```

Session struct:

```go
type LoginSession struct {
  ID string
  Status string
  Username string
  AccessToken string
  CreatedAt time.Time
}
```

Session store:

```
map[string]*LoginSession
```

---

## 4.2 Session lifecycle

Statuses:

```
pending
approved
expired
```

Tasks:

* create session
* fetch session
* approve session
* cleanup expired sessions

---

# Milestone 5 — GitHub OAuth Integration

Goal: authenticate user via GitHub.

---

## 5.1 OAuth setup

```
internal/oauth/github.go
```

Create OAuth config:

```go
oauth2.Config{
  ClientID
  ClientSecret
  RedirectURL
}
```

---

## 5.2 OAuth login redirect

Endpoint:

```
GET /login/{session_id}
```

Steps:

1 validate session
2 redirect to GitHub OAuth

---

## 5.3 OAuth callback

Endpoint:

```
GET /oauth/callback
```

Tasks:

1 exchange code for token
2 call GitHub API:

```
GET /user
```

3 retrieve username
4 verify org membership:

```
GET /user/orgs
```

5 mark session approved

---

# Milestone 6 — Server API

Goal: full API for CLI and servers.

---

## 6.1 Router

```
internal/api/router.go
```

Routes:

```
POST /api/v1/login/start
GET  /api/v1/login/status/{session_id}

GET  /api/v1/ca/pub
POST /api/v1/sign/user
POST /api/v1/sign/host

GET  /login/{session_id}
GET  /oauth/callback
```

---

## 6.2 Login start endpoint

```
POST /api/v1/login/start
```

Steps:

```
create session
return login URL
```

---

## 6.3 Login status endpoint

```
GET /api/v1/login/status/{session_id}
```

Return:

```
pending
approved
```

If approved return access token.

---

## 6.4 CA public key endpoint

```
GET /api/v1/ca/pub
```

Return:

```
ssh-ed25519 AAA...
```

---

## 6.5 Sign user certificate

```
POST /api/v1/sign/user
```

Input:

```
public_key
access_token
```

Steps:

```
verify token
extract principal
sign certificate
return cert
```

---

## 6.6 Sign host certificate

```
POST /api/v1/sign/host
```

Input:

```
host_public_key
```

Return:

```
host_certificate
```

---

# Milestone 7 — CLI Implementation (`sshifu`)

Goal: CLI obtains cert and runs ssh.

---

## 7.1 Parse CLI args

Command:

```
sshifu <sshifu-server> [ssh args]
```

Example:

```
sshifu auth.example.com target-server
```

---

## 7.2 Detect identity file

Check:

```
-i option
```

Else default:

```
~/.ssh/id_ed25519
```

---

## 7.3 Detect certificate

Look for:

```
~/.ssh/id_ed25519-cert.pub
```

Check expiration.

If valid:

```
skip login
```

---

## 7.4 Login flow

1 start login

```
POST /login/start
```

2 print login URL

3 poll:

```
GET /login/status
```

until approved.

---

## 7.5 Request certificate

```
POST /sign/user
```

Send:

```
public key
access token
```

Save result:

```
~/.ssh/id_ed25519-cert.pub
```

---

## 7.6 Download CA

Call:

```
GET /ca/pub
```

Append to:

```
~/.ssh/known_hosts
```

Entry:

```
@cert-authority * ssh-ed25519 ...
```

---

## 7.7 Run SSH

Execute:

```go
exec.Command("ssh", args...)
```

Add:

```
-o CertificateFile=<cert>
```

---

# Milestone 8 — Server Tool (`sshifu-trust`)

Goal: configure SSH servers.

---

## 8.1 Download CA

```
GET /ca/pub
```

Write:

```
/etc/ssh/sshifu_ca.pub
```

---

## 8.2 Retrieve host key

Read:

```
/etc/ssh/ssh_host_ed25519_key.pub
```

---

## 8.3 Request host certificate

```
POST /sign/host
```

---

## 8.4 Install certificate

Write:

```
/etc/ssh/ssh_host_ed25519_key-cert.pub
```

---

## 8.5 Update sshd config

Add:

```
TrustedUserCAKeys /etc/ssh/sshifu_ca.pub
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub
```

---

## 8.6 Restart sshd

```
systemctl restart sshd
```

---

# Milestone 9 — End-to-End Testing

Environment:

```
1 sshifu-server
1 ssh server
1 client
```

Test flow:

```
sshifu auth.example.com target-server
```

Expected:

```
browser login
certificate issued
ssh connection success
```

---

# Milestone 10 — Hardening

Add:

### Validation

* reject invalid public keys
* validate cert TTL
* enforce OAuth org membership

### CLI improvements

* error handling
* logging
* timeouts

---

# Recommended Development Order

This sequence minimizes debugging pain:

1️⃣ Config + CA
2️⃣ Certificate signing
3️⃣ API server skeleton
4️⃣ Login session system
5️⃣ GitHub OAuth
6️⃣ CLI login
7️⃣ SSH execution
8️⃣ sshifu-trust

---

# Expected Code Size

Approximate:

| Component     | LOC |
| ------------- | --- |
| sshifu CLI    | 400 |
| sshifu-server | 800 |
| sshifu-trust  | 200 |

Total:

```
~1400 LOC
```


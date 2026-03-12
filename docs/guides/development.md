# Development Guide

This guide covers the Sshifu architecture, building from source, and contributing.

## Table of Contents

- [Project Overview](#project-overview)
- [Architecture](#architecture)
- [Development Setup](#development-setup)
- [Building](#building)
- [Testing](#testing)
- [Code Structure](#code-structure)
- [Contributing](#contributing)

---

## Project Overview

**Sshifu** (SSH + 師傅 "master") is a lightweight SSH authentication system using short-lived OpenSSH certificates with OAuth authentication.

**Design goals:**
- Minimal infrastructure (single server, no database)
- Standard OpenSSH compatibility
- Simple OAuth integration (GitHub organizations)
- Designed for small teams (<50 users)

---

## Architecture

### System Components

```
┌─────────────────┐
│  User CLI       │  sshifu
│  (sshifu)       │
└────────┬────────┘
         │
         │ 1. Start login session
         │ 2. Poll for approval
         │ 3. Request certificate
         ▼
┌─────────────────────────┐
│  sshifu-server          │
│  - OAuth gateway        │
│  - SSH Certificate Auth │
└─────────────────────────┘
         │
         │ Configure trust
         ▼
┌─────────────────────────┐
│  Target SSH Server      │
│  (configured via        │
│   sshifu-trust)         │
└─────────────────────────┘
```

### Authentication Flow

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
```

---

## Development Setup

### Requirements

- **Go 1.25+**
- **Git**
- **OpenSSH 6.7+** (for testing)

### Clone the Repository

```bash
git clone https://github.com/azophy/sshifu.git
cd sshifu
```

### Install Dependencies

```bash
go mod download
```

### Project Structure

```
sshifu/
├── cmd/
│   ├── sshifu/          # User CLI
│   ├── sshifu-server/   # Server component
│   └── sshifu-trust/    # Server setup tool
│
├── internal/
│   ├── api/             # HTTP API handlers
│   ├── cert/            # SSH certificate operations
│   ├── config/          # Configuration loading
│   ├── oauth/           # OAuth provider implementations
│   ├── session/         # Login session management
│   └── ssh/             # SSH utilities
│
├── web/                 # Web frontend (login pages)
├── e2e/                 # End-to-end tests
├── docs/                # Documentation
├── config.example.yml   # Example configuration
└── go.mod
```

---

## Building

### Build All Components

```bash
# User CLI
go build -o sshifu ./cmd/sshifu

# Server
go build -o sshifu-server ./cmd/sshifu-server

# Trust tool
go build -o sshifu-trust ./cmd/sshifu-trust
```

### Build for Production

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o sshifu-server ./cmd/sshifu-server

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o sshifu ./cmd/sshifu
```

### Run During Development

```bash
# Run server with auto-reload (using air or similar)
go run ./cmd/sshifu-server

# Run CLI
go run ./cmd/sshifu auth.example.com user@target
```

---

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests by Package

```bash
# Config tests
go test ./internal/config/...

# Certificate tests
go test ./internal/cert/...

# API tests
go test ./internal/api/...

# OAuth tests
go test ./internal/oauth/...
```

### Run Tests with Coverage

```bash
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### End-to-End Tests

```bash
# Run E2E tests
go test ./e2e/...

# Run with verbose output
go test -v ./e2e/...
```

### Test Environment Setup

For E2E tests, you need:

1. A running sshifu-server instance
2. A test SSH server
3. Test GitHub OAuth credentials

Create a test configuration:

```yaml
# test-config.yml
server:
  listen: ":8080"
  public_url: http://localhost:8080

ca:
  private_key: ./test_ca
  public_key: ./test_ca.pub

cert:
  ttl: 1h

auth:
  providers:
    - name: github
      type: github
      client_id: test_client_id
      client_secret: test_client_secret
      allowed_org: test-org
```

### Testing the Install Script

The install script (`scripts/install.sh`) supports multiple platforms and configurations:

```bash
# Test on Linux (all apps)
./scripts/install.sh

# Test single app installation
INSTALL_APP=sshifu ./scripts/install.sh

# Test multiple apps
INSTALL_APP=sshifu,sshifu-server ./scripts/install.sh

# Test specific version
INSTALL_VERSION=0.6.3 ./scripts/install.sh

# Test custom prefix
INSTALL_PREFIX=/tmp/sshifu-test ./scripts/install.sh

# Test with CLI flags
./scripts/install.sh --app sshifu --version 0.6.3 --prefix /tmp/sshifu-test

# Test verbose output
INSTALL_VERBOSE=1 ./scripts/install.sh

# Test no PATH modification
INSTALL_NO_PATH=1 ./scripts/install.sh

# Test help
./scripts/install.sh --help
```

**Platform-specific notes:**
- Linux/macOS: Downloads `.tar.gz` archives
- Windows (WSL/Git Bash): Downloads `.zip` archives
- Architecture auto-detection: `amd64`, `arm64`, `arm`

---

## Code Structure

### cmd/

Entry points for each binary:

| Directory | Binary | Purpose |
|-----------|--------|---------|
| `cmd/sshifu/` | `sshifu` | User CLI |
| `cmd/sshifu-server/` | `sshifu-server` | Server |
| `cmd/sshifu-trust/` | `sshifu-trust` | Server setup tool |

### internal/

Core packages:

| Package | Purpose |
|---------|---------|
| `api/` | HTTP handlers and API endpoints |
| `cert/` | SSH CA operations, certificate signing |
| `config/` | YAML config loading and validation |
| `oauth/` | OAuth provider implementations |
| `session/` | Login session storage and management |
| `ssh/` | SSH key utilities |

### web/

Static assets:

| File | Purpose |
|------|---------|
| `login.html` | OAuth login page |

---

## Key Packages

### cert/ - Certificate Authority

Handles SSH certificate signing:

```go
// Load CA from private key file
func LoadCA(privateKeyPath string) (*CA, error)

// Sign a user certificate
func (ca *CA) SignUserKey(
    userKey ssh.PublicKey,
    principal string,
    ttl time.Duration,
) ([]byte, error)

// Sign a host certificate
func (ca *CA) SignHostKey(
    hostKey ssh.PublicKey,
    principals []string,
    ttl time.Duration,
) ([]byte, error)
```

### session/ - Session Management

In-memory session storage:

```go
type LoginSession struct {
    ID          string
    Status      string  // "pending", "approved", "expired"
    Username    string
    AccessToken string
    CreatedAt   time.Time
}

type Store struct {
    sessions  map[string]*LoginSession
    timeout   time.Duration
}

func (s *Store) Create(id string) *LoginSession
func (s *Store) Get(id string) *LoginSession
func (s *Store) Approve(id, username, token string)
func (s *Store) Cleanup()
```

### oauth/ - OAuth Providers

GitHub OAuth implementation:

```go
type Provider interface {
    AuthCodeURL(state string) string
    Exchange(code string) (*oauth2.Token, error)
    GetUserInfo(token *oauth2.Token) (*UserInfo, error)
}

type UserInfo struct {
    Username string
    Orgs     []string
}
```

---

## Contributing

### Development Workflow

1. **Fork the repository**

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature
   ```

3. **Make changes and write tests**

4. **Run tests**
   ```bash
   go test ./...
   ```

5. **Run linter** (if configured)
   ```bash
   go vet ./...
   ```

6. **Commit with clear messages**
   ```bash
   git commit -m "feat: add new feature"
   ```

7. **Push and create PR**

### Commit Message Format

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

### Code Style

- Follow Go standard formatting (`go fmt`)
- Use meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

### Testing Guidelines

- Write unit tests for new packages
- Add integration tests for API endpoints
- Include E2E tests for critical flows
- Aim for >80% coverage on new code

---

## Debugging

### Enable Debug Logging

Add logging to your code:

```go
log.Printf("DEBUG: Processing request for user %s", username)
```

### Test OAuth Flow Locally

1. Set up ngrok for local tunneling:
   ```bash
   ngrok http 8080
   ```

2. Update GitHub OAuth callback URL to ngrok URL

3. Run server locally:
   ```bash
   go run ./cmd/sshifu-server
   ```

### Inspect Certificates

```bash
# View certificate details
ssh-keygen -L -f id_ed25519-cert.pub

# Verify certificate signature
ssh-keygen -V -f ca.pub id_ed25519-cert.pub
```

---

## Release Process

1. **Update version** in relevant files

2. **Run all tests**
   ```bash
   go test ./...
   ```

3. **Build binaries**
   ```bash
   GOOS=linux GOARCH=amd64 go build -o sshifu-linux-amd64 ./cmd/sshifu
   GOOS=darwin GOARCH=arm64 go build -o sshifu-darwin-arm64 ./cmd/sshifu
   ```

4. **Create git tag**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

5. **Create GitHub release** with binaries

---

## Next Steps

- [API Reference](../api/README.md) - API documentation
- [Configuration Reference](../reference/configuration.md) - Config options
- [User Guide](guides/user-guide.md) - End user documentation

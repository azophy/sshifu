# Sshifu - Agent Guidelines

## Project Overview

**Sshifu** is a lightweight SSH authentication system using short-lived OpenSSH certificates with OAuth (GitHub organizations).

**Tech Stack:**
- **Language:** Go 1.25+
- **Key Dependencies:** `golang.org/x/crypto`, `golang.org/x/oauth2`, `github.com/goccy/go-yaml`
- **Components:** 3 binaries (`sshifu`, `sshifu-server`, `sshifu-trust`)

## Quick Commands

```bash
# Build all
go build ./cmd/...

# Run tests
go test ./...

# Run server locally
go run ./cmd/sshifu-server

# Run CLI
go run ./cmd/sshifu auth.example.com user@target
```

## Project Structure

```
sshifu/
├── cmd/              # Binary entry points
│   ├── sshifu/       # User CLI
│   ├── sshifu-server/
│   └── sshifu-trust/
├── internal/         # Core packages
│   ├── api/          # HTTP handlers
│   ├── cert/         # SSH CA operations
│   ├── config/       # YAML config loading
│   ├── oauth/        # OAuth providers
│   ├── session/      # Session management
│   └── ssh/          # SSH utilities
├── web/              # Static assets
└── docs/             # Documentation
```

## Architecture

```
User CLI → sshifu-server (OAuth + CA) → Target SSH Server
```

**Auth Flow:** CLI starts session → User OAuth via GitHub → Server verifies org membership → Certificate issued → SSH connects

## Key Conventions

- **Go standard formatting** (`go fmt`)
- **Meaningful names**, small focused functions
- **Comments for complex logic only**
- **Unit + integration + E2E tests** for critical flows
- **No long-term secrets** on clients; short-lived certs only

## Configuration

Server uses `config.yml`:

```yaml
server:
  listen: ":8080"
  public_url: https://auth.example.com
ca:
  private_key: ./ca
  public_key: ./ca.pub
cert:
  ttl: 8h
auth:
  providers:
    - name: github
      type: github
      client_id: ...
      client_secret: ...
      allowed_org: my-org
```

## Documentation

- [`docs/README.md`](docs/README.md) - Overview
- [`docs/guides/development.md`](docs/guides/development.md) - Build/contribute
- [`docs/api/README.md`](docs/api/README.md) - API reference
- [`docs/reference/configuration.md`](docs/reference/configuration.md) - Config options

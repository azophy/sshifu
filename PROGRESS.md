# Sshifu Progress

## Completed Milestones

### Milestone 1 — Project Bootstrap ✅

**Status:** Complete  
**Date:** March 8, 2026

**What was done:**

1. **Initialized Go module**
   - Module path: `github.com/azophy/sshifu`

2. **Created project directory structure**
   ```
   cmd/
     sshifu/
     sshifu-server/
     sshifu-trust/
   internal/
     config/
     cert/
     session/
     oauth/
     api/
     ssh/
   web/
   ```

3. **Created placeholder binaries**
   - `cmd/sshifu/main.go` - CLI client entry point
   - `cmd/sshifu-server/main.go` - Server entry point
   - `cmd/sshifu-trust/main.go` - Trust configuration tool entry point

4. **Implemented internal packages**
   - `internal/config/config.go` - YAML configuration loader with validation
   - `internal/cert/ca.go` - SSH CA for signing user/host certificates
   - `internal/session/session.go` - In-memory login session management with cleanup
   - `internal/oauth/github.go` - GitHub OAuth provider implementation
   - `internal/api/response.go` - API response types and helpers
   - `internal/ssh/key.go` - SSH key/certificate utilities

5. **Created web assets**
   - `web/login.html` - OAuth login page with status polling

6. **Configuration**
   - `config.example.yml` - Example configuration template

7. **Dependencies installed**
   - `github.com/goccy/go-yaml`
   - `golang.org/x/crypto`
   - `golang.org/x/oauth2`

**Verification:**
- ✅ All packages build successfully
- ✅ No compile errors

---

## Pending Milestones

### Milestone 2 — Config + Setup Wizard
### Milestone 3 — SSH Certificate Authority
### Milestone 4 — Login Session System
### Milestone 5 — GitHub OAuth Integration
### Milestone 6 — Server API
### Milestone 7 — CLI Implementation
### Milestone 8 — Server Tool
### Milestone 9 — End-to-End Testing
### Milestone 10 — Hardening

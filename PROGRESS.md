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

### Milestone 2 — Config + Setup Wizard ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Implemented setup wizard** (`internal/config/wizard.go`)
   - Interactive prompts for server configuration
   - Asks for: server public URL, CA key path, GitHub OAuth credentials, allowed org
   - Auto-generates CA keypair if not present
   - Saves configuration to `config.yml`

2. **Updated sshifu-server main.go**
   - Checks for existing config on startup
   - Launches wizard if config not found
   - Displays configuration summary on startup

3. **Added helper functions to config package**
   - `Save()` - saves config to YAML file
   - `Marshal()` - marshals config to YAML
   - `GenerateCAKeys()` - generates ED25519 CA keypair

**Verification:**
- ✅ Wizard starts when config.yml is missing
- ✅ Config loads successfully when present
- ✅ All binaries build without errors

---

### Milestone 3 — SSH Certificate Authority ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Created user certificate signing** (`internal/cert/user_cert.go`)
   - `SignUserKey()` function for signing user SSH certificates
   - Supports configurable TTL (time-to-live)
   - Supports certificate extensions (permit-pty, permit-port-forwarding, etc.)
   - Applies sensible defaults when no extensions specified

2. **Created host certificate signing** (`internal/cert/host_cert.go`)
   - `SignHostKey()` function for signing host SSH certificates
   - Supports multiple principals (hostnames/IPs)
   - Configurable TTL for host certificates

3. **Refactored CA struct** (`internal/cert/ca.go`)
   - Updated `CA.SignUserKey()` to delegate to standalone `SignUserKey()`
   - Updated `CA.SignHostKey()` to delegate to standalone `SignHostKey()`
   - Maintains backward-compatible API for CA struct users

4. **Added comprehensive unit tests** (`internal/cert/cert_test.go`)
   - `TestGenerateCA` - CA keypair generation
   - `TestLoadCAInvalidKey` - Error handling for invalid keys
   - `TestLoadCANonexistentKey` - Error handling for missing files
   - `TestSignUserKey` - User certificate signing and verification
   - `TestSignUserKeyDefaultExtensions` - Default extension application
   - `TestSignHostKey` - Host certificate signing and verification
   - `TestSignHostKeyEmptyPrincipals` - Edge case handling
   - `TestCA_Methods` - CA struct method integration tests

**Verification:**
- ✅ All 8 unit tests pass
- ✅ Build succeeds with no errors
- ✅ Certificate signing works correctly for both user and host certificates

---

### Milestone 4 — Login Session System ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

The login session system was already implemented in Milestone 1 (`internal/session/session.go`). This milestone is considered complete as the session management functionality is fully functional:

1. **Session Store** (`internal/session/session.go`)
   - In-memory session storage using `map[string]*LoginSession`
   - Session statuses: `pending`, `approved`, `expired`
   - Automatic cleanup of expired sessions via background goroutine
   - Configurable session TTL (default 15 minutes)

2. **Session Operations**
   - `Create()` - creates new pending session
   - `Get()` - retrieves session (returns expired sessions as non-existent)
   - `Approve()` - marks session as approved with username and access token

3. **Added comprehensive unit tests** (`internal/session/session_test.go`)
   - `TestNewStore` - store initialization
   - `TestNewStoreDefaultMaxAge` - default TTL validation
   - `TestCreateAndGet` - session creation and retrieval
   - `TestGetNonExistent` - error handling for missing sessions
   - `TestApprove` - session approval workflow
   - `TestApproveNonExistent` - error handling for invalid sessions
   - `TestSessionExpiration` - TTL enforcement
   - `TestIsExpired` - expiration check logic
   - `TestCleanup` - automatic cleanup of expired sessions

**Verification:**
- ✅ All 9 unit tests pass
- ✅ Session expiration works correctly
- ✅ Automatic cleanup removes expired sessions

---

### Milestone 5 — GitHub OAuth Integration ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Implemented API handlers** (`internal/api/handlers.go`)
   - `LoginStart` - POST `/api/v1/login/start` - creates new login session, returns session ID and login URL
   - `LoginStatus` - GET `/api/v1/login/status/{id}` - returns session status (pending/approved) with access token
   - `Login` - GET `/login/{id}` - displays OAuth login page with status polling
   - `OAuthInit` - GET `/oauth/github/{id}` - initiates GitHub OAuth flow
   - `OAuthCallback` - GET `/oauth/callback` - handles OAuth callback, exchanges code for token, verifies org membership, approves session
   - `CAPublicKey` - GET `/api/v1/ca/pub` - returns CA public key for SSH known_hosts
   - `SignUserCertificate` - POST `/api/v1/sign/user` - placeholder for certificate signing (Milestone 6)
   - `SignHostCertificate` - POST `/api/v1/sign/host` - placeholder for host cert signing (Milestone 8)

2. **Updated sshifu-server main.go**
   - Initialize session store with 15-minute TTL
   - Initialize GitHub OAuth provider from configuration
   - Load CA public key for distribution
   - Setup HTTP router with all API and OAuth routes
   - Start HTTP server on configured listen address

3. **Added comprehensive unit tests**
   - `internal/oauth/github_test.go` - 6 tests for GitHub OAuth provider
     - `TestNewGitHubProvider` - provider initialization
     - `TestGitHubProviderAuthURL` - OAuth URL generation
     - `TestGitHubProviderExchangeInvalidCode` - error handling
     - `TestGitHubProviderGetUsernameInvalidToken` - error handling
     - `TestGitHubProviderVerifyMembershipNoOrg` - org verification bypass
     - `TestGitHubProviderVerifyMembershipInvalidToken` - error handling
   - `internal/api/handlers_test.go` - 7 tests for API handlers
     - `TestLoginStart` - session creation endpoint
     - `TestLoginStartWrongMethod` - HTTP method validation
     - `TestLoginStatus` - pending session status
     - `TestLoginStatusApproved` - approved session with access token
     - `TestLoginStatusNotFound` - 404 for missing sessions
     - `TestCAPublicKey` - CA public key endpoint
     - `TestLoginTemplateExists` - login page rendering

**Verification:**
- ✅ All 22 new unit tests pass (9 session + 6 oauth + 7 api handlers)
- ✅ Build succeeds with no errors
- ✅ Total test count: 31 tests across all packages

---

## Pending Milestones

### Milestone 6 — Server API
### Milestone 7 — CLI Implementation
### Milestone 8 — Server Tool
### Milestone 9 — End-to-End Testing
### Milestone 10 — Hardening

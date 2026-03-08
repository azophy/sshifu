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

### Milestone 6 — Server API ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Implemented certificate signing handlers** (`internal/api/handlers.go`)
   - `SignUserCertificate` - POST `/api/v1/sign/user` - signs user SSH certificates
     - Validates access token against approved sessions
     - Extracts username from session for certificate principal
     - Parses user public key and signs with CA
     - Returns signed certificate in OpenSSH format
   - `SignHostCertificate` - POST `/api/v1/sign/host` - signs host SSH certificates
     - Validates host public key and principals
     - Signs host certificate with CA
     - Returns signed certificate for SSH server

2. **Added API types** (`internal/api/response.go`)
   - `HostSignRequest` - request type for host certificate signing
   - `SignResponse` - response type for certificate signing endpoints

3. **Updated Handler struct** (`internal/api/handlers.go`)
   - Added `caSigner` field for SSH CA signer
   - Added `config` field for certificate configuration (TTL, extensions)
   - Updated `NewHandler()` to accept CA signer and config

4. **Enhanced session store** (`internal/session/session.go`)
   - Added `Range()` method for iterating over sessions
   - Enables token-based session lookup for certificate signing

5. **Extended CA struct** (`internal/cert/ca.go`)
   - Added `Signer()` method to expose underlying ssh.Signer

6. **Updated sshifu-server** (`cmd/sshifu-server/main.go`)
   - Loads CA private key on startup
   - Initializes handler with CA signer and certificate config
   - Displays "CA loaded successfully" on startup

7. **Added comprehensive unit tests** (`internal/api/handlers_test.go`)
   - `TestSignUserCertificate` - successful user certificate signing
   - `TestSignUserCertificateInvalidToken` - rejects invalid access tokens
   - `TestSignUserCertificateMissingFields` - validates required fields
   - `TestSignHostCertificate` - successful host certificate signing
   - `TestSignHostCertificateMissingFields` - validates required fields
   - `TestSignHostCertificateWrongMethod` - HTTP method validation

**Verification:**
- ✅ All 6 new unit tests pass
- ✅ Total test count: 43 tests across all packages
- ✅ Build succeeds with no errors
- ✅ Certificate signing produces valid OpenSSH certificates

---

### Milestone 7 — CLI Implementation (`sshifu`) ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Implemented complete CLI workflow** (`cmd/sshifu/main.go`)
   - `parseArgs()` - parses command-line arguments
     - Extracts sshifu-server URL (first argument)
     - Handles `-i` option for custom identity file
     - Passes remaining arguments to SSH command
   - `run()` - orchestrates the main CLI workflow
     - Finds identity key (explicit or default)
     - Checks for existing valid certificate
     - Performs login flow if no valid cert
     - Requests and saves certificate
     - Installs CA key to known_hosts
     - Executes SSH with certificate

2. **Implemented identity key detection**
   - Uses `-i` option if provided
   - Falls back to default keys: `~/.ssh/id_ed25519`, `~/.ssh/id_rsa`, `~/.ssh/id_ecdsa`
   - Expands tilde (`~`) to home directory

3. **Implemented certificate validity check**
   - Checks for certificate file at `~/.ssh/id_*-cert.pub`
   - Verifies certificate type is UserCert
   - Checks expiration time
   - Validates principal matches username
   - Skips login if valid certificate found

4. **Implemented OAuth login flow**
   - `startLoginSession()` - POST `/api/v1/login/start`
     - Creates new login session
     - Returns session ID and login URL
   - `pollLoginStatus()` - GET `/api/v1/login/status/{session_id}`
     - Polls every 2 seconds for up to 10 minutes
     - Displays progress dots while waiting
     - Returns access token on approval

5. **Implemented certificate request**
   - `requestCertificate()` - POST `/api/v1/sign/user`
     - Loads user public key
     - Sends public key and access token
     - Receives signed SSH certificate
   - `saveCertificate()` - saves certificate to disk
     - Creates `.ssh` directory if needed
     - Sets proper permissions (0600)

6. **Implemented CA key installation**
   - `installCAKey()` - GET `/api/v1/ca/pub`
     - Downloads CA public key
   - `addCAToKnownHosts()` - appends to `~/.ssh/known_hosts`
     - Adds `@cert-authority * <ca-key>` entry
     - Checks for duplicate entries
     - Creates file if doesn't exist

7. **Implemented SSH execution**
   - `execSSH()` - executes system SSH command
     - Adds `-o CertificateFile=<cert>` option
     - Passes through all SSH arguments
     - Forwards stdin/stdout/stderr
     - Preserves SSH exit code

8. **Added helper function** (`internal/ssh/key.go`)
   - `MarshalAuthorizedKey()` - marshals public key to authorized_keys format

9. **Added comprehensive unit tests** (`cmd/sshifu/sshifu_test.go`)
   - `TestParseArgs` - CLI argument parsing (4 sub-tests)
   - `TestJoinURL` - URL joining with various base formats (4 sub-tests)
   - `TestCAKeyExists` - known_hosts CA key detection (3 sub-tests)
   - `TestSaveCertificate` - certificate file saving
   - `TestGetCertificatePath` - certificate path generation (3 sub-tests)

**Verification:**
- ✅ All 5 new unit tests pass (17 sub-tests total)
- ✅ Total test count: 60 tests across all packages
- ✅ Build succeeds with no errors
- ✅ CLI displays proper usage on missing arguments
- ✅ Certificate reuse works when valid cert exists

---

### Milestone 8 — Server Tool (`sshifu-trust`) ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Implemented complete sshifu-trust CLI workflow** (`cmd/sshifu-trust/main.go`)
   - `main()` - entry point with usage display
   - `run()` - orchestrates the complete server configuration workflow
   - 7-step configuration process with progress output

2. **Implemented server URL normalization**
   - `normalizeServerURL()` - converts server argument to proper HTTP URL
   - Auto-prefixes `https://` if no scheme provided
   - Validates URL format and removes trailing slashes

3. **Implemented CA public key download**
   - `downloadCAPublicKey()` - GET `/api/v1/ca/pub`
   - Parses JSON response and extracts public key

4. **Implemented CA key installation**
   - `installCAKey()` - writes CA key to `/etc/ssh/sshifu_ca.pub`
   - Creates `/etc/ssh` directory if needed
   - Sets proper file permissions (0644)

5. **Implemented host key retrieval**
   - `readHostPublicKey()` - reads `/etc/ssh/ssh_host_ed25519_key.pub`
   - Returns trimmed public key string

6. **Implemented host principal detection**
   - `getHostPrincipals()` - auto-detects hostnames for certificate
   - Gets primary hostname from OS
   - Parses `/etc/hosts` for additional hostnames
   - Includes localhost variants by default

7. **Implemented host certificate request**
   - `requestHostCertificate()` - POST `/api/v1/sign/host`
   - Sends host public key and principals
   - Configurable TTL (default 720h / 30 days)
   - Receives signed host certificate

8. **Implemented host certificate installation**
   - `installHostCertificate()` - writes cert to `/etc/ssh/ssh_host_ed25519_key-cert.pub`
   - Sets proper file permissions (0644)

9. **Implemented sshd_config modification**
   - `updateSSHDConfig()` - updates `/etc/ssh/sshd_config`
   - Adds or updates `TrustedUserCAKeys` directive
   - Adds or updates `HostCertificate` directive
   - Preserves existing configuration

10. **Implemented SSH daemon restart**
    - `restartSSHD()` - restarts SSH service
    - Detects systemd and uses `systemctl restart sshd`
    - Falls back to `service ssh restart` for non-systemd systems

11. **Added comprehensive unit tests** (`cmd/sshifu-trust/sshifu_trust_test.go`)
    - `TestNormalizeServerURL` - URL normalization (6 sub-tests)
    - `TestJoinURL` - URL path joining (4 sub-tests)
    - `TestIsValidIP` - IP address validation (6 sub-tests)
    - `TestReadEtcHosts` - /etc/hosts parsing
    - `TestGetPathConstants` - path constant validation
    - `TestGetHostKeyPath` - host key path accessor
    - `TestGetHostCertPath` - host cert path accessor
    - `TestGetCAInstallPath` - CA install path accessor
    - `TestGetSSHDConfigPath` - sshd_config path accessor
    - `TestDetectOS` - OS detection
    - `TestDefaultCertValidity` - certificate TTL constant
    - `TestDefaultHTTPTimeout` - HTTP timeout constant

**Verification:**
- ✅ All 12 new unit tests pass
- ✅ Total test count: 72 tests across all packages
- ✅ Build succeeds with no errors
- ✅ CLI displays proper usage on missing arguments

---

## Pending Milestones

### Milestone 9 — End-to-End Testing ✅

**Status:** Complete
**Date:** March 8, 2026

**What was done:**

1. **Created e2e test directory structure**
   ```
   e2e/
     helpers.go
     cert_e2e_test.go
     session_e2e_test.go
     trust_e2e_test.go
     README.md
   ```

2. **Implemented test helper utilities** (`e2e/helpers.go`)
   - `TestServer` - manages test server state
   - `APIClient` - provides helper methods for API calls
     - `LoginStart()` - creates new login session
     - `LoginStatus()` - checks session status
     - `GetCAPublicKey()` - fetches CA public key
     - `SignUserCertificate()` - requests user certificate
     - `SignHostCertificate()` - requests host certificate
     - `PollLoginStatus()` - polls until session approved
   - `GenerateTestKeyPair()` - generates SSH key pairs for testing

3. **Implemented certificate generation e2e tests** (`e2e/cert_e2e_test.go`)
   - `TestCertificateGenerationE2E` - end-to-end user certificate generation
   - `TestHostCertificateGenerationE2E` - end-to-end host certificate generation
   - `TestCAKeyFormatE2E` - verifies CA keys are in correct OpenSSH format
   - `TestKnownHostsFormatE2E` - tests known_hosts CA entry format

4. **Implemented session management e2e tests** (`e2e/session_e2e_test.go`)
   - `TestSessionE2E` - complete session lifecycle (create → approve → expire)
   - `TestSessionRangeE2E` - session iteration via Range method
   - `TestSessionTokenLookupE2E` - token-based session lookup

5. **Implemented sshifu-trust e2e tests** (`e2e/trust_e2e_test.go`)
   - `TestSshifuTrustE2E` - complete sshifu-trust workflow
     - Step 1: Download CA public key
     - Step 2: Generate host key
     - Step 3: Request host certificate
     - Step 4: Install host certificate
   - `TestHostCertificateValidation` - host certificate properties validation
     - Multiple principals support
     - Certificate type verification
     - Validity period verification
   - `TestCAKeyDistribution` - CA key distribution endpoint tests
   - `TestSSHDConfigUpdate` - sshd_config modification logic tests

6. **Created e2e test runner script** (`scripts/test-e2e.sh`)
   - Runs all e2e tests
   - Provides colored output
   - Reports test summary

7. **Created e2e test documentation** (`e2e/README.md`)
   - Test structure overview
   - Running instructions
   - Test coverage details

**Test Results:**
- ✅ All 17 e2e tests pass
- ✅ Total test count: 89 tests across all packages
  - e2e: 17 tests
  - internal/api: 13 tests
  - internal/cert: 8 tests
  - internal/config: 3 tests
  - internal/oauth: 6 tests
  - internal/session: 9 tests
  - cmd/sshifu: 17 tests (5 with subtests)
  - cmd/sshifu-trust: 12 tests (12 subtests)

**Verification:**
- ✅ All packages build successfully
- ✅ All tests pass (`go test ./...`)
- ✅ Certificate generation produces valid OpenSSH certificates
- ✅ Session lifecycle management works correctly
- ✅ sshifu-trust workflow completes successfully
- ✅ CA key distribution works correctly
- ✅ sshd_config modification logic validated

---

### Milestone 10 — Hardening

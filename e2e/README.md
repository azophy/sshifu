# Sshifu E2E Tests

This directory contains end-to-end tests for the Sshifu system.

## Test Structure

- `helpers.go` - Common test utilities and API client
- `server_e2e_test.go` - Tests for sshifu-server components
- `cli_e2e_test.go` - Tests for sshifu CLI workflow
- `trust_e2e_test.go` - Tests for sshifu-trust workflow

## Running Tests

### Run all e2e tests

```bash
./scripts/test-e2e.sh
```

Or directly:

```bash
go test -v ./e2e/...
```

### Run specific test file

```bash
go test -v ./e2e -run TestServerE2E
go test -v ./e2e -run TestCLIE2E
go test -v ./e2e -run TestSshifuTrustE2E
```

### Run in short mode (skip e2e)

```bash
go test -short ./...
```

## Test Coverage

The e2e tests cover:

### Server Tests
- Login session creation and management
- OAuth flow simulation
- Certificate signing (user and host)
- CA public key distribution
- Session lifecycle (pending → approved → expired)

### CLI Tests
- Argument parsing
- Certificate request workflow
- CA key installation
- Certificate reuse validation

### Trust Tests
- CA key download and installation
- Host key generation
- Host certificate request
- sshd_config modification

## Test Environment

Tests use:
- In-memory session storage
- Mock OAuth provider (no real GitHub calls)
- httptest.Server for HTTP testing
- Temporary directories for file operations

## Requirements

- Go 1.21+
- Standard Go testing framework

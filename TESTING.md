# Testing Guide

## 1. Local Workspace Testing

### Test workspace installation
```bash
# From project root
npm install

# Verify symlinks created
ls -la node_modules/ | grep sshifu
```

### Test CLI wrappers
```bash
# Build Go binaries first
go build -o bin/sshifu ./cmd/sshifu
go build -o bin/sshifu-server ./cmd/sshifu-server
go build -o bin/sshifu-trust ./cmd/sshifu-trust

# Copy to package bin directories
cp bin/sshifu packages/sshifu/bin/
cp bin/sshifu-server packages/sshifu-server/bin/
cp bin/sshifu-trust packages/sshifu-trust/bin/

# Test via npx
npx sshifu --help
npx sshifu-server --help
npx sshifu-trust --help
```

## 2. Test Binary Download (Simulated Release)

### Create a test release archive manually
```bash
# Build a test binary
go build -o sshifu ./cmd/sshifu

# Create test archive
tar -czvf sshifu-linux-amd64.tar.gz sshifu

# Move to a test directory
mkdir -p /tmp/sshifu-test
mv sshifu-linux-amd64.tar.gz /tmp/sshifu-test/
cd /tmp/sshifu-test

# Test installation from local package
npm init -y
npm install /home/azophy/DEV/sshifu/packages/sshifu
```

### Or test with a dev version override
```bash
# In packages/sshifu/package.json, temporarily set a real version
# e.g., "version": "0.4.0"
# Then run npm install to see if it downloads from GitHub
```

## 3. Test Version Sync Script

```bash
# Test version sync
node scripts/sync-version.js 1.0.0-test

# Verify all versions updated
cat package.json | jq .version
cat packages/sshifu/package.json | jq .version
cat packages/sshifu-server/package.json | jq .version
cat packages/sshifu-trust/package.json | jq .version

# Reset to dev version
node scripts/sync-version.js 0.0.0-dev
```

## 4. Test Build Script

```bash
# Build all binaries (creates dist/ folder)
node scripts/build-all-binaries.js 1.0.0-test

# Check output
ls -la dist/

# Expected: 15 archives (3 binaries × 5 platforms)
# sshifu-linux-amd64.tar.gz
# sshifu-linux-arm64.tar.gz
# sshifu-darwin-amd64.tar.gz
# sshifu-darwin-arm64.tar.gz
# sshifu-windows-amd64.zip
# (same for sshifu-server and sshifu-trust)
```

## 5. Test GitHub Actions Workflow (Dry Run)

### Use act (local GitHub Actions runner)
```bash
# Install act: https://nektosact.com/introduction.html
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# List available workflows
act -l

# Run release workflow locally (dry run)
act release -n

# Or run with a fake release event
act release --eventpath test-release-event.json
```

### Create test event payload
```json
// test-release-event.json
{
  "action": "published",
  "release": {
    "tag_name": "v1.0.0-test"
  }
}
```

## 6. Test npm Publish (Dry Run)

```bash
# Check what would be published
npm publish --dry-run --workspace=packages/sshifu
npm publish --dry-run --workspace=packages/sshifu-server
npm publish --dry-run --workspace=packages/sshifu-trust

# Or all at once
npm publish --dry-run --workspaces
```

## 7. End-to-End Test (Clean Environment)

```bash
# Create a clean test directory
mkdir -p /tmp/sshifu-e2e-test
cd /tmp/sshifu-e2e-test

# Test installing from npm (after publishing a test version)
npm install sshifu
npm install sshifu-server
npm install sshifu-trust

# Verify binaries exist
ls -la node_modules/sshifu/bin/
ls -la node_modules/sshifu-server/bin/
ls -la node_modules/sshifu-trust/bin/

# Test execution
./node_modules/sshifu/bin/sshifu.js --help
./node_modules/sshifu-server/bin/sshifu-server.js --help
./node_modules/sshifu-trust/bin/sshifu-trust.js --help
```

## 8. Test with GitHub Test Release

### Create a draft release on GitHub
1. Go to https://github.com/azophy/sshifu/releases
2. Create a draft release with tag `v0.5.0-test`
3. Upload test binaries manually
4. Publish the release

### Test installation
```bash
# In a clean directory
npm init -y
npm install sshifu@0.5.0-test
```

## Quick Test Checklist

- [ ] `npm install` works in project root
- [ ] `npx sshifu --help` works (with local binary)
- [ ] `node scripts/sync-version.js 1.0.0` updates all packages
- [ ] `node scripts/build-all-binaries.js 1.0.0` creates all archives
- [ ] `npm publish --dry-run --workspaces` shows correct packages
- [ ] GitHub Actions workflow syntax is valid (use YAML linter)

# Release Workflow

## Version Synchronization

All three packages (`sshifu`, `sshifu-server`, `sshifu-trust`) share the same version number.

## Release Steps

### 1. Build Binaries for All Platforms

```bash
node scripts/build-all-binaries.js 1.2.3
```

This creates archives in `dist/`:
- `sshifu-linux-amd64.tar.gz`
- `sshifu-linux-arm64.tar.gz`
- `sshifu-darwin-amd64.tar.gz`
- `sshifu-darwin-arm64.tar.gz`
- `sshifu-windows-amd64.zip`
- (and same for sshifu-server and sshifu-trust)

### 2. Create GitHub Release

1. Go to https://github.com/azophy/sshifu/releases
2. Create a new tag: `v1.2.3`
3. Upload all artifacts from `dist/`
4. Publish the release

### 3. Bump Versions

```bash
node scripts/sync-version.js 1.2.3
```

This updates:
- `packages/sshifu/package.json`
- `packages/sshifu-server/package.json`
- `packages/sshifu-trust/package.json`

### 4. Commit and Push

```bash
git add packages/*/package.json package.json
git commit -m "chore: release v1.2.3"
git push
```

### 5. Publish to npm

```bash
# Publish all three packages at once
npm publish --workspaces --access public
```

> **Note:** You need to be logged in to npm (`npm login`) and have publish access to all three packages.

## Development Workflow

For local development:

```bash
# Build Go binaries
go build ./cmd/sshifu
go build ./cmd/sshifu-server
go build ./cmd/sshifu-trust

# Copy to package bin directories
cp sshifu packages/sshifu/bin/
cp sshifu-server packages/sshifu-server/bin/
cp sshifu-trust packages/sshifu-trust/bin/

# Test with npx
npx sshifu --help
npx sshifu-server --help
npx sshifu-trust --help
```

## Package Contents

Each npm package contains:
- `bin/<name>.js` - Node.js wrapper script
- `bin/<binary>` - The Go binary (downloaded on install)
- `scripts/install.js` - Postinstall script
- `package.json` - Package metadata

The install script downloads the appropriate binary for the user's platform from GitHub Releases.

# NPM Packages

This directory contains the npm packages for sshifu. Each package downloads only its corresponding binary.

## Packages

| Package | Description |
|---------|-------------|
| [sshifu](./sshifu/) | User CLI for SSH authentication and connection |
| [sshifu-server](./sshifu-server/) | Server component with OAuth gateway and CA |
| [sshifu-trust](./sshifu-trust/) | Server-side tool to configure SSH trust |

## Version Management

All packages share the same version. To bump versions:

```bash
# Bump all packages to a specific version
node scripts/sync-version.js 1.2.3

# Commit the version change
git add packages/*/package.json package.json
git commit -m "chore: bump version to 1.2.3"

# Publish all packages
npm publish --workspaces --access public
```

## Building Binaries for Release

```bash
# Build all binaries for all platforms
node scripts/build-all-binaries.js 1.2.3

# This creates archives in dist/:
# - sshifu-linux-amd64.tar.gz
# - sshifu-server-linux-amd64.tar.gz
# - sshifu-trust-linux-amd64.tar.gz
# - ... (for all platforms)
```

## Development

```bash
# Install all workspace packages
npm install

# Build Go binaries locally
npm run build

# Link packages for local testing
npm link ./packages/sshifu
npm link ./packages/sshifu-server
npm link ./packages/sshifu-trust
```

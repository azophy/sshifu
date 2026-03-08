# npm Package Setup Summary

## What Was Created

### Files Added

1. **package.json** - npm package configuration
   - Defines package name, version, and metadata
   - Specifies three binary commands: `sshifu`, `sshifu-server`, `sshifu-trust`
   - Sets up postinstall script to download platform-specific binaries
   - Lists supported OS and CPU architectures

2. **bin/sshifu.js** - Wrapper script for sshifu CLI
3. **bin/sshifu-server.js** - Wrapper script for sshifu-server
4. **bin/sshifu-trust.js** - Wrapper script for sshifu-trust

5. **scripts/install.js** - Postinstall script that:
   - Detects current platform (OS + architecture)
   - Fetches latest version from GitHub API
   - Downloads correct binary archive from GitHub Releases
   - Extracts binaries to bin/ directory
   - Makes binaries executable

6. **.npmignore** - Excludes unnecessary files from npm package
   - Only includes wrapper scripts and install script (6.6 kB total)
   - Binaries are downloaded on-demand for the user's platform

7. **docs/PUBLISHING.md** - Guide for publishing to npm

### Files Modified

1. **.github/workflows/release.yml**
   - Added `linux-arm64` to build matrix
   - Added `publish-npm` job to automatically publish to npm on release

2. **README.md**
   - Added npm installation as the recommended option
   - Shows both `npm install -g` and `npx` usage

3. **.gitignore**
   - Added entries to ignore downloaded binaries in bin/

## How It Works

### User Experience

```bash
# Install globally
npm install -g sshifu

# Or run without installation
npx sshifu auth.example.com user@target.com
```

### Installation Flow

1. User runs `npm install -g sshifu`
2. npm executes `scripts/install.js` as postinstall hook
3. Install script detects platform: `linux-x64` → `linux-amd64`
4. Downloads `sshifu-linux-amd64.tar.gz` from GitHub Releases
5. Extracts all three binaries to `bin/` directory
6. User can now run: `sshifu`, `sshifu-server`, `sshifu-trust`

### Platform Support

| Platform | Architectures |
|----------|---------------|
| Linux | x64, arm64, arm |
| macOS | x64 (Intel), arm64 (Apple Silicon) |
| Windows | x64 |

## Publishing

### Automatic (Recommended)

1. Create GitHub release with tag like `v1.0.0`
2. GitHub Actions workflow:
   - Builds binaries for all platforms
   - Uploads to GitHub Releases
   - Publishes to npm with same version

### Manual Testing

```bash
# Test package contents
npm pack --dry-run

# Create package file
npm pack

# Install locally to test
npm install -g ./sshifu-*.tgz

# Test commands
sshifu --help
sshifu-server --help
sshifu-trust --help
```

### First Publish

```bash
# Login to npm
npm login

# Publish (first time requires --access public)
npm publish --access public
```

## Next Steps

1. **Create npm account** (if you don't have one):
   - Go to https://www.npmjs.com/
   - Create account

2. **Generate npm token**:
   ```bash
   npm login
   ```

3. **Add NPM_TOKEN to GitHub secrets**:
   - Go to repository Settings → Secrets and variables → Actions
   - Add secret `NPM_TOKEN` with your npm token

4. **Test locally**:
   ```bash
   npm pack
   npm install -g ./sshifu-*.tgz
   sshifu --help
   ```

5. **Publish first version**:
   - Create GitHub release
   - Workflow will auto-publish to npm

## Package Size

- **npm package**: 6.6 kB (compressed)
- **Downloaded binaries**: ~30-35 MB (uncompressed, platform-dependent)
- **Installation time**: ~5-10 seconds (depends on internet speed)

## Benefits

✅ **Single package** - One npm package for all three binaries
✅ **Multi-platform** - Automatically fetches correct binary for OS/arch
✅ **Small package size** - Only 6.6 kB, binaries downloaded on-demand
✅ **Works with npx** - Can run without installation: `npx sshifu`
✅ **Automatic updates** - Users get latest version with `npm update`
✅ **No Go required** - End users don't need Go installed

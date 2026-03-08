# Publishing to npm

This document describes how to publish sshifu to npm.

## Prerequisites

1. **npm account**: You need an npm account with publish permissions
2. **NPM_TOKEN**: Set up `NPM_TOKEN` secret in GitHub repository settings

## Setup NPM Token

1. Create an npm account at https://www.npmjs.com/
2. Generate an access token:
   ```bash
   npm login
   ```
3. Go to GitHub repository settings → Secrets and variables → Actions
4. Add a new secret named `NPM_TOKEN` with your npm token

## Publishing

### Automatic (Recommended)

The package is automatically published when you create a GitHub release:

1. Create a new release on GitHub
2. The release workflow will:
   - Build binaries for all platforms
   - Upload binaries as release assets
   - Publish the npm package

### Manual

To publish manually:

```bash
# Update version in package.json
npm version <major|minor|patch|prerelease>

# Build the package
npm pack

# Publish to npm
npm publish --access public
```

## How It Works

### Package Structure

```
sshifu/
├── bin/
│   ├── sshifu.js          # Node.js wrapper for sshifu binary
│   ├── sshifu-server.js   # Node.js wrapper for sshifu-server binary
│   └── sshifu-trust.js    # Node.js wrapper for sshifu-trust binary
├── scripts/
│   └── install.js         # Postinstall script that downloads binaries
├── package.json           # npm package configuration
└── README.md              # Documentation
```

### Installation Flow

1. User runs `npm install -g sshifu`
2. npm runs the `postinstall` script (`scripts/install.js`)
3. The install script:
   - Detects the current platform (OS + architecture)
   - Fetches the latest version from GitHub API
   - Downloads the appropriate binary archive from GitHub Releases
   - Extracts binaries to the `bin/` directory
   - Makes binaries executable (Unix/Mac)

### Platform Mapping

| npm Platform | Go OS/Arch | Archive Name |
|--------------|------------|--------------|
| linux-x64 | linux-amd64 | sshifu-linux-amd64.tar.gz |
| linux-arm64 | linux-arm64 | sshifu-linux-arm64.tar.gz |
| linux-arm | linux-arm | sshifu-linux-arm.tar.gz |
| darwin-x64 | darwin-amd64 | sshifu-darwin-amd64.tar.gz |
| darwin-arm64 | darwin-arm64 | sshifu-darwin-arm64.tar.gz |
| win32-x64 | windows-amd64 | sshifu-windows-amd64.zip |

## Supported Platforms

- **Linux**: x64, arm64, arm
- **macOS**: Intel (x64), Apple Silicon (arm64)
- **Windows**: x64

## Usage

After installation:

```bash
# Global installation
sshifu auth.example.com user@target.com
sshifu-server
sshifu-trust auth.example.com

# Run without installation
npx sshifu auth.example.com user@target.com
```

## Troubleshooting

### Postinstall fails

If the postinstall script fails, users can:

1. Check internet connection
2. Verify GitHub Releases exist for the version
3. Manually download binaries from https://github.com/azophy/sshifu/releases

### Wrong binary downloaded

The install script maps npm platform names to Go platform names. Check the mapping in `scripts/install.js`.

### Binary not found after install

Run:
```bash
npm rebuild sshifu
```

## Testing Locally

Test the package before publishing:

```bash
# Create a test package
npm pack

# Install locally
npm install -g ./sshifu-*.tgz

# Test commands
sshifu --help
sshifu-server --help
sshifu-trust --help
```

## Version Management

The package version is managed through:

1. **GitHub Releases**: Version is extracted from the release tag
2. **package.json**: Updated automatically by the release workflow

For manual publishing, use:
```bash
npm version <major|minor|patch|prerelease>
```

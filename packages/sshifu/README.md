# sshifu

SSH authentication client with short-lived certificates and OAuth authentication (GitHub organizations).

## Installation

### Quick Start (Recommended)

Use `npx` to run without installing:

```bash
npx sshifu auth.example.com user@target-server.com
```

### Install Globally

For frequent use, install globally:

```bash
npm install -g sshifu
```

Then use directly:

```bash
sshifu auth.example.com user@target-server.com
```

## Features

- 🔐 **Short-lived SSH certificates** - Automatic certificate issuance with configurable TTL (default 8 hours)
- 🌐 **GitHub OAuth authentication** - Authenticate users via GitHub organization membership
- 🛠️ **Standard OpenSSH compatibility** - Works with existing `ssh` command without workflow changes

## Requirements

- OpenSSH 6.7+ (for certificate support)
- Node.js 14.0.0+

## Full Documentation

See the complete documentation at [github.com/azophy/sshifu](https://github.com/azophy/sshifu)

## License

MIT

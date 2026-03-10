# sshifu-server

SSH authentication server with OAuth gateway and certificate authority.

## Installation

```bash
npm install -g sshifu-server
```

## Usage

```bash
sshifu-server
```

The server will prompt you to configure:
- GitHub OAuth credentials
- Certificate Authority (CA) settings
- Server listen address

## Features

- 🔐 **SSH Certificate Authority** - Issues short-lived SSH certificates
- 🌐 **OAuth Gateway** - GitHub organization authentication
- 📦 **Minimal Infrastructure** - Single server, no database required

## Requirements

- Node.js 14.0.0+
- Go binary (will be downloaded automatically on install)

## Full Documentation

See the complete documentation at [github.com/azophy/sshifu](https://github.com/azophy/sshifu)

## License

MIT

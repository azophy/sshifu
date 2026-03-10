# sshifu-trust

Configure SSH servers to trust the sshifu certificate authority.

## Installation

### Quick Start (Recommended)

Use `npx` to run without installing:

```bash
sudo npx sshifu-trust auth.example.com
```

### Install Globally

For frequent use, install globally:

```bash
npm install -g sshifu-trust
```

Then use directly:

```bash
sudo sshifu-trust auth.example.com
```

This will configure your SSH server to trust certificates signed by the sshifu CA at the specified address.

## Features

- 🔐 **CA Trust Configuration** - Automatically configures SSH server to trust the CA
- 🛠️ **Simple Setup** - One command to configure trust
- 🔒 **Secure** - Only trusts the specific CA, no other changes

## Requirements

- Node.js 14.0.0+
- sudo access on the target SSH server
- Go binary (will be downloaded automatically on install)

## Full Documentation

See the complete documentation at [github.com/azophy/sshifu](https://github.com/azophy/sshifu)

## License

MIT

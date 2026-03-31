# sshifu-trust (Bash Script)

Lightweight bash script to configure SSH servers to trust the sshifu Certificate Authority.

## Why Bash?

Target SSH servers often don't have Node.js or Go installed. This bash script requires only standard Linux utilities (`bash`, `curl`) and works on any modern Linux distribution.

## Quick Start

### Option 1: Run directly with curl (Recommended)

```bash
# One-liner with sudo
curl -fsSL https://raw.githubusercontent.com/azophy/sshifu/main/cmd/sshifu-trust/sshifu-trust.sh | sudo bash -s -- auth.example.com

# Or with environment variable
curl -fsSL https://raw.githubusercontent.com/azophy/sshifu/main/cmd/sshifu-trust/sshifu-trust.sh | SSHIFU_SERVER=auth.example.com sudo bash
```

### Option 2: Download and run

```bash
# Download the script
curl -fsSL https://raw.githubusercontent.com/azophy/sshifu/main/cmd/sshifu-trust/sshifu-trust.sh -o /tmp/sshifu-trust.sh

# Make executable
chmod +x /tmp/sshifu-trust.sh

# Run (requires sudo)
sudo /tmp/sshifu-trust.sh auth.example.com
```

## Usage

```bash
sudo sshifu-trust.sh [options] <sshifu-server>
```

### Arguments

| Argument | Description |
|----------|-------------|
| `<sshifu-server>` | URL or hostname of the sshifu server (e.g., `auth.example.com`) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `SSHIFU_SERVER` | sshifu server URL (used if no argument provided) |

### Commands

| Command | Description |
|---------|-------------|
| `help`, `-h`, `--help` | Show help message |
| `version`, `-v`, `--version` | Show version information |

### Examples

```bash
# Basic usage with hostname
sudo sshifu-trust.sh auth.example.com

# With full URL
sudo sshifu-trust.sh https://auth.example.com

# With custom port
sudo sshifu-trust.sh auth.example.com:8080

# Using environment variable
SSHIFU_SERVER=auth.example.com sudo sshifu-trust.sh

# Interactive mode (will prompt for server)
sudo sshifu-trust.sh
```

## What It Does

The script performs the following steps:

1. **Download CA public key** from sshifu-server
2. **Install CA key** to `/etc/ssh/sshifu_ca.pub`
3. **Read host public key** from `/etc/ssh/ssh_host_ed25519_key.pub`
4. **Get host principals** (hostname, localhost variants)
5. **Request host certificate** from sshifu-server
6. **Install host certificate** to `/etc/ssh/ssh_host_ed25519_key-cert.pub`
7. **Update sshd_config** to trust the CA and use the certificate
8. **Restart SSH daemon**

## Requirements

- **bash** 4.0+
- **curl**
- **sudo** access
- **OpenSSH** 6.7+
- **systemd** or **sysvinit**

## Testing

Run the test suite:

```bash
./sshifu-trust-test.sh
```

## Troubleshooting

### "This script must be run as root"

The script requires sudo/root access to modify SSH configuration. Run with:

```bash
sudo sshifu-trust.sh auth.example.com
```

### "Failed to download CA public key"

Check that:
- The sshifu-server is running and accessible
- The URL/hostname is correct
- Network connectivity is working

Test manually:

```bash
curl -v https://auth.example.com/api/v1/ca/pub
```

### "Host public key not found"

The script expects an ED25519 host key at `/etc/ssh/ssh_host_ed25519_key.pub`. Generate one if missing:

```bash
ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ""
```

### "Failed to restart SSH service"

The script tries multiple service names (`sshd`, `ssh`, `openssh-daemon`, `openssh`). Check which service name your system uses:

```bash
systemctl status sshd
# or
systemctl status ssh
```

## Files Modified

| File | Purpose |
|------|---------|
| `/etc/ssh/sshifu_ca.pub` | CA public key |
| `/etc/ssh/ssh_host_ed25519_key-cert.pub` | Host certificate |
| `/etc/ssh/sshd_config` | SSH daemon configuration |

## Security Notes

- The script only modifies SSH configuration to trust the specific CA
- No long-term secrets are stored on the target server
- Host certificates expire after 30 days by default
- The CA private key never leaves the sshifu-server

## License

MIT License - See [LICENSE](../../LICENSE)

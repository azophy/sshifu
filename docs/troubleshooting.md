# Troubleshooting Guide

Common issues and solutions for Sshifu.

## Table of Contents

- [User CLI Issues](#user-cli-issues)
- [Server Issues](#server-issues)
- [OAuth Issues](#oauth-issues)
- [SSH Connection Issues](#ssh-connection-issues)
- [Certificate Issues](#certificate-issues)
- [FAQ](#faq)

---

## User CLI Issues

### "No valid certificate found" on every run

**Symptoms:**
```
No valid certificate found, starting login flow...
```

**Causes:**
1. Certificate expired (normal after TTL)
2. Certificate file missing or corrupted
3. Wrong identity key path

**Solutions:**

1. Check if certificate exists:
   ```bash
   ls -la ~/.ssh/id_ed25519-cert.pub
   ```

2. Verify certificate validity:
   ```bash
   ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub
   ```

3. If using custom key, ensure correct path:
   ```bash
   sshifu auth.example.com -i ~/.ssh/custom_key user@server
   ```

---

### "Login timeout" error

**Symptoms:**
```
Waiting for authentication...
Error: login timeout
```

**Causes:**
1. OAuth flow not completed in browser
2. Network connectivity issues
3. Server not accessible

**Solutions:**

1. Ensure you opened the login URL in browser
2. Check network connectivity to server:
   ```bash
   curl https://auth.example.com/api/v1/ca/pub
   ```

3. Verify server is running and accessible

4. Try again with verbose output (if available)

---

### "Failed to find identity key"

**Symptoms:**
```
Error: failed to find identity key: ...
```

**Causes:**
1. Default SSH key doesn't exist
2. Custom key path is incorrect
3. Key file permissions issue

**Solutions:**

1. Check for existing keys:
   ```bash
   ls -la ~/.ssh/id_*
   ```

2. Generate a new key if needed:
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519
   ```

3. Specify correct key path:
   ```bash
   sshifu auth.example.com -i ~/.ssh/correct_key user@server
   ```

4. Fix permissions:
   ```bash
   chmod 600 ~/.ssh/id_ed25519
   ```

---

### "Certificate request failed"

**Symptoms:**
```
Error: failed to request certificate: ...
```

**Causes:**
1. Server returned error
2. Invalid access token
3. Session expired

**Solutions:**

1. Check server logs for details
2. Try logging in again (certificate will be reissued)
3. Verify server is accessible

---

## Server Issues

### Server fails to start

**Symptoms:**
```
Failed to load configuration: ...
```

**Causes:**
1. Invalid YAML syntax
2. Missing required fields
3. File permissions issue

**Solutions:**

1. Validate YAML syntax:
   ```bash
   python3 -c "import yaml; yaml.safe_load(open('config.yml'))"
   ```

2. Check required fields:
   - `server.public_url`
   - `auth.providers` (at least one)

3. Verify file permissions:
   ```bash
   chmod 600 config.yml
   chmod 600 ca  # CA private key
   ```

---

### "No supported OAuth provider configured"

**Symptoms:**
```
Fatal: No supported OAuth provider configured
```

**Causes:**
1. No providers in config
2. Invalid provider type
3. Missing OAuth credentials

**Solutions:**

1. Add provider to config:
   ```yaml
   auth:
     providers:
       - name: github
         type: github
         client_id: YOUR_CLIENT_ID
         client_secret: YOUR_CLIENT_SECRET
         allowed_org: your-org
   ```

2. Verify provider type is supported (`github` or `oidc`)

3. Ensure all required fields are present

---

### "Failed to load CA private key"

**Symptoms:**
```
Fatal: Failed to load CA private key: ...
```

**Causes:**
1. Key file doesn't exist
2. Invalid key format
3. Permissions issue

**Solutions:**

1. Check if key exists:
   ```bash
   ls -la ./ca
   ```

2. If missing, delete config to trigger regeneration:
   ```bash
   rm config.yml
   ./sshifu-server  # Will run wizard
   ```

3. Fix permissions:
   ```bash
   chmod 600 ./ca
   ```

---

### Server starts but OAuth fails

**Symptoms:**
- Server starts successfully
- OAuth redirect fails
- "Invalid redirect_uri" from GitHub

**Causes:**
1. `public_url` doesn't match OAuth app settings
2. Using http vs https mismatch
3. Callback URL misconfigured

**Solutions:**

1. Verify `public_url` in config:
   ```yaml
   server:
     public_url: https://auth.example.com
   ```

2. Check GitHub OAuth App settings:
   - Authorization callback URL must be: `{public_url}/oauth/callback`

3. Ensure protocol matches (http vs https)

---

## OAuth Issues

### "Organization membership verification failed"

**Symptoms:**
- OAuth succeeds
- But session not approved
- User is member of org but still fails

**Causes:**
1. Wrong organization in config
2. User's org membership is private
3. OAuth app not authorized for org

**Solutions:**

1. Verify `allowed_org` matches GitHub org name exactly

2. Check org membership visibility:
   - User must have public org membership, OR
   - OAuth app must be authorized by org

3. Re-authorize OAuth app for the organization

---

### "Invalid client_id or client_secret"

**Symptoms:**
```
OAuth exchange failed: bad_verification_code
```

**Causes:**
1. Wrong credentials in config
2. Credentials copied incorrectly
3. Secret was regenerated

**Solutions:**

1. Verify credentials in GitHub OAuth App settings

2. Re-copy Client ID and Client Secret

3. If secret was regenerated, update config and restart server

---

### Login URL returns 404

**Symptoms:**
- Login URL shows 404 error
- `/login/{session_id}` not found

**Causes:**
1. Server not running
2. Wrong base URL
3. Session expired before URL opened

**Solutions:**

1. Verify server is running:
   ```bash
   curl https://auth.example.com/api/v1/ca/pub
   ```

2. Check `public_url` matches actual server URL

3. Generate new login session (sessions expire)

---

## SSH Connection Issues

### "Permission denied (publickey)"

**Symptoms:**
```
user@server: Permission denied (publickey).
```

**Causes:**
1. CA not trusted by target server
2. User not in authorized principals
3. Certificate not presented

**Solutions:**

1. Run `sshifu-trust` on target server:
   ```bash
   sudo sshifu-trust auth.example.com
   ```

2. Verify CA key installed:
   ```bash
   cat /etc/ssh/sshifu_ca.pub
   ```

3. Check sshd_config:
   ```bash
   grep TrustedUserCAKeys /etc/ssh/sshd_config
   ```

4. Verify certificate is being used:
   ```bash
   ssh -v -o CertificateFile=~/.ssh/id_ed25519-cert.pub user@server
   ```

---

### "Certificate signed by untrusted CA"

**Symptoms:**
```
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@       WARNING: UNTRUSTED CA CERTIFICATE                 @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
```

**Causes:**
1. CA key not in known_hosts
2. CA key changed

**Solutions:**

1. Let sshifu install CA key automatically, OR

2. Manually add to known_hosts:
   ```bash
   echo "@cert-authority * $(curl -s https://auth.example.com/api/v1/ca/pub | jq -r '.public_key')" >> ~/.ssh/known_hosts
   ```

---

### SSH connection hangs

**Symptoms:**
- Connection established but hangs
- No prompt appears

**Causes:**
1. DNS resolution issues
2. SSH daemon not responding
3. Network/firewall issues

**Solutions:**

1. Try with verbose SSH:
   ```bash
   ssh -v user@server
   ```

2. Check DNS resolution:
   ```bash
   dig server.example.com
   ```

3. Verify SSH daemon is running:
   ```bash
   sudo systemctl status sshd
   ```

---

## Certificate Issues

### Certificate expires too quickly

**Symptoms:**
- Certificate expires before expected
- Frequent re-authentication required

**Causes:**
1. Short TTL in server config
2. System clock skew

**Solutions:**

1. Increase TTL in server config:
   ```yaml
   cert:
     ttl: 24h  # Increase from default 8h
   ```

2. Check system clock synchronization:
   ```bash
   timedatectl status
   ```

---

### Certificate not reusable

**Symptoms:**
- Certificate exists but not used
- Login required every time

**Causes:**
1. Certificate expired
2. Wrong certificate file
3. Certificate principal mismatch

**Solutions:**

1. Check certificate validity:
   ```bash
   ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub
   ```

2. Verify certificate matches key:
   ```bash
   ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub | grep "Signing CA"
   ```

3. Ensure using correct identity key

---

## FAQ

### How long do certificates last?

Default is 8 hours, configurable via `cert.ttl` in server config.

### Can I use multiple SSH keys?

Yes, specify with `-i` option:
```bash
sshifu auth.example.com -i ~/.ssh/work_key user@server
```

### Do I need to run sshifu-trust on every server?

Yes, each target SSH server must be configured to trust the CA.

### What happens if I lose the CA private key?

All issued certificates become invalid. You'll need to:
1. Generate new CA
2. Re-run sshifu-trust on all servers
3. Users re-authenticate

### Can I revoke a certificate?

Not in v1. Certificates expire automatically based on TTL.

### Is my GitHub password stored?

No. Sshifu only receives an OAuth access token, not your password.

### Can I use Sshifu with personal GitHub accounts?

Yes, as long as you're a member of the configured organization.

### Does Sshifu work with GitHub Enterprise?

Not in v1. Currently supports GitHub.com only.

### Can I use Sshifu behind a corporate proxy?

Yes, configure proxy in your environment:
```bash
export https_proxy=http://proxy.company.com:8080
```

### What SSH key types are supported?

- ed25519 (recommended)
- RSA
- ECDSA

### Can I limit which servers users can access?

Not in v1. Any authenticated user can access any configured server. Access control is handled by the target server's OS (users, groups, etc.).

---

## Getting Help

If you're still experiencing issues:

1. **Check logs:**
   - Server: `journalctl -u sshifu-server -f`
   - CLI: Run with verbose mode if available

2. **Verify setup:**
   - Server accessible: `curl https://auth.example.com/api/v1/ca/pub`
   - CA trusted: `cat /etc/ssh/sshifu_ca.pub`
   - Certificate valid: `ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub`

3. **Report a bug:**
   - Include error messages
   - Describe steps to reproduce
   - Include relevant logs

---

## Next Steps

- [User Guide](../guides/user-guide.md) - Usage documentation
- [Server Setup Guide](../guides/server-setup.md) - Deployment guide
- [Configuration Reference](configuration.md) - All config options

# OAuth Provider Configuration Guide

This guide explains how to configure and use OAuth providers with sshifu-server, including GitHub and OIDC providers.

## Overview

sshifu-server supports multiple OAuth providers simultaneously:

- **GitHub** - Authenticate via GitHub organization membership
- **OIDC** - Authenticate via any OpenID Connect-compatible provider (Google, Okta, Auth0, Keycloak, etc.)

Users can choose their preferred provider at login, and each provider can have different authorization rules.

---

## How OAuth Works in Sshifu

### Authentication Flow

```
┌─────────┐     ┌──────────────┐     ┌─────────┐     ┌──────────────┐
│  User   │     │  sshifu CLI  │     │  Server │     │ OAuth Provider│
│  SSH    │     │              │     │         │     │  (GitHub/OIDC)│
└────┬────┘     └──────┬───────┘     └────┬────┘     └───────┬────────┘
     │                 │                   │                  │
     │ ssh user@host   │                   │                  │
     │────────────────>│                   │                  │
     │                 │                   │                  │
     │ POST /login/start                   │                  │
     │────────────────>│                   │                  │
     │                 │                   │                  │
     │ login_url       │                   │                  │
     │<────────────────│                   │                  │
     │                 │                   │                  │
     │ Open browser    │                   │                  │
     │────────────────────────────────────>│                  │
     │                 │                   │                  │
     │                 │                   │  Authorize       │
     │                 │                   │─────────────────>│
     │                 │                   │                  │
     │                 │                   │  code + state    │
     │                 │                   │<─────────────────│
     │                 │                   │                  │
     │                 │                   │ Exchange code    │
     │                 │                   │─────────────────>│
     │                 │                   │                  │
     │                 │                   │ User info        │
     │                 │                   │<─────────────────│
     │                 │                   │                  │
     │                 │   Poll status     │                  │
     │<────────────────────────────────────────────────────────│
     │                 │                   │                  │
     │                 │ Issue certificate│                  │
     │<────────────────│                   │                  │
     │                 │                   │                  │
     │ SSH with cert   │                   │                  │
     │───────────────────────────────────────────────────────>│
```

### State Parameter Encoding

The OAuth `state` parameter encodes both the provider name and session ID:

```
state = "{provider_name}:{session_id}"
```

**Example:** `github:abc123def456`

This allows the callback handler to:
1. Identify which provider to use for token exchange
2. Retrieve the correct session
3. Avoid trying multiple providers (efficient and reliable)

---

## GitHub Provider

### Setup Steps

1. **Create GitHub OAuth App**

   Go to your GitHub organization settings:
   ```
   https://github.com/organizations/<your-org>/settings/developer_settings
   ```

2. **Register New OAuth App**

   Fill in:
   - **Application name**: `sshifu` (or descriptive name)
   - **Homepage URL**: Your sshifu-server public URL
   - **Authorization callback URL**: `{public_url}/oauth/callback`

3. **Get Credentials**

   - Copy **Client ID**
   - Click **Generate a new client secret** and copy it

4. **Configure sshifu-server**

   ```yaml
   auth:
     providers:
       - name: github
         type: github
         client_id: Ov23liEXAMPLE123
         client_secret: abc123def456ghi789jkl012mno345pqr678
         allowed_org: your-github-org
   ```

### Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Provider identifier (shown in UI) |
| `type` | Yes | Must be `"github"` |
| `client_id` | Yes | GitHub OAuth Client ID |
| `client_secret` | Yes | GitHub OAuth Client Secret |
| `allowed_org` | Yes | GitHub organization for membership check |

### Authorization

Users must be members of the configured `allowed_org`. Membership is verified on each login.

---

## OIDC Provider

### Setup Steps

1. **Choose OIDC Provider**

   Common providers:
   - Google Workspace
   - Okta
   - Auth0
   - Keycloak (self-hosted)
   - Microsoft Entra ID (Azure AD)

2. **Register OAuth Client**

   In your OIDC provider's admin console:
   - Create new OAuth/OIDC client
   - Set redirect URI: `{public_url}/oauth/callback`
   - Note Client ID and Client Secret
   - Note the OIDC Issuer URL

3. **Configure sshifu-server**

   ```yaml
   auth:
     providers:
       - name: google
         type: oidc
         issuer: https://accounts.google.com
         client_id: YOUR_CLIENT_ID.apps.googleusercontent.com
         client_secret: YOUR_CLIENT_SECRET
         principal_oauth_field_name: email
   ```

### Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Provider identifier (shown in UI) |
| `type` | Yes | Must be `"oidc"` |
| `issuer` | Yes | OIDC issuer URL |
| `client_id` | Yes | OIDC Client ID |
| `client_secret` | Yes | OIDC Client Secret |
| `principal_oauth_field_name` | No | Field to extract as username (default: `preferred_username`) |

### Automatic Configuration Discovery

sshifu-server automatically discovers OIDC endpoints from the issuer URL:

```
GET {issuer}/.well-known/openid-configuration
```

Returns:
```json
{
  "issuer": "https://accounts.google.com",
  "authorization_endpoint": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_endpoint": "https://oauth2.googleapis.com/token",
  "userinfo_endpoint": "https://openidconnect.googleapis.com/v1/userinfo",
  ...
}
```

### Username Extraction

The `principal_oauth_field_name` specifies which OIDC claim to use as the SSH username:

```yaml
# Use email (without @domain)
principal_oauth_field_name: email

# Use preferred username (default)
principal_oauth_field_name: preferred_username

# Use subject ID
principal_oauth_field_name: sub

# Use custom claim
principal_oauth_field_name: custom_username
```

### Provider-Specific Examples

#### Google Workspace

```yaml
auth:
  providers:
    - name: google
      type: oidc
      issuer: https://accounts.google.com
      client_id: YOUR_CLIENT_ID.apps.googleusercontent.com
      client_secret: YOUR_CLIENT_SECRET
      principal_oauth_field_name: email
```

#### Okta

```yaml
auth:
  providers:
    - name: okta
      type: oidc
      issuer: https://your-org.okta.com/oauth2/default
      client_id: 0oa1234567890abcdef
      client_secret: YOUR_CLIENT_SECRET
      principal_oauth_field_name: preferred_username
```

#### Auth0

```yaml
auth:
  providers:
    - name: auth0
      type: oidc
      issuer: https://your-domain.auth0.com
      client_id: YOUR_CLIENT_ID
      client_secret: YOUR_CLIENT_SECRET
      principal_oauth_field_name: nickname
```

#### Keycloak

```yaml
auth:
  providers:
    - name: keycloak
      type: oidc
      issuer: https://keycloak.example.com/realms/myrealm
      client_id: sshifu
      client_secret: YOUR_CLIENT_SECRET
      principal_oauth_field_name: preferred_username
```

---

## Multiple Providers

Configure multiple providers to give users choice:

```yaml
auth:
  providers:
    # Primary GitHub org
    - name: github
      type: github
      client_id: Ov23liEXAMPLE123
      client_secret: secret123
      allowed_org: company-main

    # Secondary GitHub org
    - name: github-contractors
      type: github
      client_id: Ov23liEXAMPLE456
      client_secret: secret456
      allowed_org: company-contractors

    # Google Workspace
    - name: google
      type: oidc
      issuer: https://accounts.google.com
      client_id: CLIENT_ID.apps.googleusercontent.com
      client_secret: secret789
      principal_oauth_field_name: email
```

### Login Experience

When multiple providers are configured:
1. User runs `sshifu auth.example.com user@target`
2. CLI displays login URL
3. User opens browser and sees **all configured providers** as buttons
4. User clicks their preferred provider
5. OAuth flow completes with selected provider

### Use Cases

- **Multiple GitHub orgs**: Main company + contractors
- **Hybrid identity**: GitHub for devs, Google for ops
- **Migrations**: Run old and new providers in parallel
- **Redundancy**: Backup auth method

---

## Callback URL Configuration

**All providers use the same callback URL:**

```
{server.public_url}/oauth/callback
```

**Example:** If `public_url: https://auth.example.com`, then callback is:
```
https://auth.example.com/oauth/callback
```

Configure this exact URL in all your OAuth provider settings.

### Why One Callback Works

The `state` parameter encodes the provider name, so the callback handler knows which provider to use:

1. User clicks "Sign in with Google"
2. State = `google:abc123`
3. Callback receives `state=google:abc123`
4. Handler extracts `provider=google`, `session=abc123`
5. Uses Google provider for token exchange

---

## Security Considerations

### Client Secrets

- Never commit secrets to version control
- Use environment variables or secret management
- Rotate secrets periodically
- Use different secrets per environment

### Redirect URIs

- Always use HTTPS in production
- Exact match required (no wildcards)
- Don't use `http://` except for local development

### Token Handling

- Access tokens are short-lived (session duration)
- Tokens are stored in memory (not persisted)
- Tokens are used only for user info retrieval

### Session Security

- Sessions expire after 15 minutes of inactivity
- Session IDs are cryptographically random
- State parameter prevents CSRF attacks

---

## Troubleshooting

### "Unknown OAuth provider" Error

**Cause:** Provider name in config doesn't match

**Fix:** Check provider `name` field is unique and valid

### Callback URL Mismatch

**Symptoms:** OAuth provider returns "redirect_uri_mismatch" error

**Fix:** 
1. Verify `server.public_url` is correct
2. Ensure callback URL in OAuth provider settings matches exactly
3. Check for trailing slashes

### OIDC Discovery Fails

**Symptoms:** Server startup fails with OIDC configuration error

**Fix:**
1. Verify `issuer` URL is correct
2. Test discovery endpoint manually:
   ```bash
   curl https://your-issuer.com/.well-known/openid-configuration
   ```
3. Check network connectivity/firewall rules

### Username Extraction Fails

**Symptoms:** "Failed to retrieve user information"

**Fix:**
1. Check `principal_oauth_field_name` exists in OIDC userinfo response
2. Test userinfo endpoint manually with token
3. Try alternative field names (`email`, `sub`, `preferred_username`)

### Multiple Providers - Wrong One Used

**Note:** This should not happen with current implementation (provider encoded in state)

**If it occurs:** Check logs for state parameter decoding errors

---

## Testing OAuth Configuration

### Test Single Provider Flow

```bash
# Start server
./sshifu-server

# In another terminal, test login
curl -X POST http://localhost:8080/api/v1/login/start

# Open the returned login_url in browser
# Complete OAuth flow
# Check logs for success/errors
```

### Test OIDC Discovery

```bash
# Test discovery endpoint
curl https://your-issuer.com/.well-known/openid-configuration | jq

# Verify required fields exist:
# - authorization_endpoint
# - token_endpoint
# - userinfo_endpoint
```

### Test Token Exchange

```bash
# After OAuth, check server logs for:
# - "✓ OAuth provider initialized"
# - Token exchange success/failure
# - Username extraction result
```

---

## Next Steps

- [Server Setup Guide](server-setup.md) - Deploy sshifu-server
- [Configuration Reference](../reference/configuration.md) - All config options
- [API Reference](../api/README.md) - OAuth API endpoints

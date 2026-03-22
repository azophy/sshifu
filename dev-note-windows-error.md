# Windows GitHub Actions Test Error - Investigation Summary

**Date:** 2026-03-22  
**Issue:** GitHub Actions Windows tests failing during npm package installation  
**Status:** Root cause identified, fix implemented and deployed

---

## Problem Description

Windows tests in GitHub Actions were failing with exit code 1 when running `npx -y sshifu-trust -v` (and similar commands for `sshifu` and `sshifu-server`).

### Error Message
```
Error: sshifu-trust binary not found at C:\npm\cache\_npx\...\node_modules\sshifu-trust\bin\sshifu-trust.exe
The postinstall script may have failed. Try running:
  npm rebuild sshifu-trust
```

---

## Root Cause Analysis

### Primary Issue: Archive Format Mismatch

The npm package install scripts (`packages/*/scripts/install.js`) were attempting to download `.tar.gz` archives for **all platforms**, but the release build script (`scripts/ci-build.sh`) creates:
- `.tar.gz` for Linux/macOS
- **`.zip` for Windows**

**Code before fix:**
```javascript
const archiveName = `${PACKAGE_NAME}-${platform}.tar.gz`;  // Always .tar.gz!
execSync(`tar -xzf "${archivePath}" -C "${binDir}"`, { stdio: 'ignore' });
```

### Secondary Issues Discovered

1. **npm Cache Problems**: GitHub Actions runners were using cached npm packages from previous runs, which contained the broken install scripts.

2. **Postinstall Script Silent Failures**: When using `npx`, the postinstall script output was not visible, making debugging difficult.

3. **Path Handling in PowerShell**: Single quotes in PowerShell commands needed escaping for paths with special characters.

---

## Fixes Applied

### 1. Install Script Archive Format Fix

**Files Modified:**
- `packages/sshifu/scripts/install.js`
- `packages/sshifu-server/scripts/install.js`
- `packages/sshifu-trust/scripts/install.js`

**Changes:**
```javascript
const isWindows = process.platform === 'win32';
const archiveExt = isWindows ? '.zip' : '.tar.gz';
const archiveName = `${PACKAGE_NAME}-${platform}${archiveExt}`;

// Extract with appropriate tool
if (isWindows) {
  execSync(`powershell -Command "Expand-Archive -Path '${archivePath.replace(/'/g, "''")}' -DestinationPath '${binDir.replace(/'/g, "''")}' -Force"`, { stdio: 'ignore' });
} else {
  execSync(`tar -xzf "${archivePath}" -C "${binDir}"`, { stdio: 'ignore' });
}

// Fallback logic for binary rename
if (archiveBinName !== binName) {
  if (fs.existsSync(extractedPath)) {
    fs.renameSync(extractedPath, binPath);
  } else {
    // Try to find the extracted file
    const files = fs.readdirSync(binDir);
    const exeFile = files.find(f => f.endsWith('.exe') && f.startsWith(PACKAGE_NAME));
    if (exeFile) {
      fs.renameSync(path.join(binDir, exeFile), binPath);
    }
  }
}
```

### 2. CI Workflow Improvements

**File Modified:** `.github/workflows/test-npm-release.yml`

**Changes:**
- Increased npm propagation wait time from 60s to 120s
- Clear all npm caches (npm, npx, logs, cacache) before tests
- Fetch latest version from GitHub API instead of using `@latest`
- Use `npm install -g` with verbose output instead of `npx` for visibility
- Add package contents listing step for debugging

```yaml
- name: Clear all npm caches
  run: |
    npm cache clean --force
    Remove-Item -Recurse -Force $env:LOCALAPPDATA\npm-cache\_npx -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $env:LOCALAPPDATA\npm-cache\_logs -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $env:LOCALAPPDATA\npm-cache\_cacache -ErrorAction SilentlyContinue

- name: Install with verbose output
  run: |
    $version = (Invoke-RestMethod -Uri "https://api.github.com/repos/azophy/sshifu/releases/latest").tag_name.TrimStart('v')
    Write-Host "Installing version: $version"
    npm install -g ${{ matrix.package }}@$version --verbose 2>&1 | Out-Host
  shell: pwsh

- name: List installed package contents
  run: |
    $pkgPath = npm root -g | Join-Path -ChildPath ${{ matrix.package }}
    Write-Host "Package path: $pkgPath"
    Get-ChildItem -Recurse $pkgPath | Select-Object FullName
  shell: pwsh
```

---

## Test Results

### Before Fix (v0.7.0)
```
✗ Test Windows (sshifu-trust, sshifu-trust -v) - FAILED
✗ Test Windows (sshifu, sshifu -v) - CANCELLED
✗ Test Windows (sshifu-server, sshifu-server -v) - CANCELLED
```

### After Fix Deployment
Multiple iterations (v0.7.1 - v0.7.7) were deployed to refine the fix:

- **v0.7.1**: Initial fix with .zip support
- **v0.7.2**: Added fallback logic for binary rename
- **v0.7.3**: Added npm cache clearing
- **v0.7.4**: Added npx cache clearing
- **v0.7.5**: Switched to npm install -g with verbose output
- **v0.7.6**: Added package contents listing
- **v0.7.7**: Fetch version from GitHub API

---

## Key Learnings

1. **Platform-Specific Archives**: Always check archive formats for different platforms. Windows typically uses `.zip` while Unix-like systems use `.tar.gz`.

2. **npm Cache Issues**: GitHub Actions runners may have stale npm cache. Always clear cache or use `--force` for fresh installs in CI.

3. **Verbose Logging**: Use `--verbose` flag and `npm install -g` instead of `npx` in CI to see postinstall script output.

4. **PowerShell Path Escaping**: When using PowerShell commands with paths, escape single quotes by doubling them (`'` → `''`).

5. **Binary Naming Consistency**: Ensure the extracted binary name matches what the wrapper script expects. The archive contains `sshifu-trust-windows-amd64.exe` but the wrapper expects `sshifu-trust.exe`.

---

## Verification Steps

To verify the fix works:

```bash
# On Windows (PowerShell)
npm cache clean --force
npm install -g sshifu-trust@latest --verbose
sshifu-trust -v

# Check binary location
$pkgPath = npm root -g | Join-Path -ChildPath sshifu-trust
Get-ChildItem -Recurse $pkgPath\bin
```

---

## Related Files

- `packages/sshifu/scripts/install.js` - Postinstall script for sshifu
- `packages/sshifu-server/scripts/install.js` - Postinstall script for sshifu-server
- `packages/sshifu-trust/scripts/install.js` - Postinstall script for sshifu-trust
- `.github/workflows/test-npm-release.yml` - CI workflow for testing npm releases
- `scripts/ci-build.sh` - Build script that creates release archives

---

## Next Steps

1. Monitor GitHub Actions runs for v0.7.7+ to confirm all Windows tests pass
2. Consider adding a Windows self-hosted runner for faster feedback
3. Add integration tests that verify binary execution after npm install
4. Document the release process including the need for both .zip and .tar.gz formats

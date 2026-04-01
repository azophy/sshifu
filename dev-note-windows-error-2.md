# Windows npm Install Fix - Development Progress

**Date:** 2026-03-22
**Issue:** Windows GitHub Actions tests failing during npm package installation
**Status:** In Progress - Root cause identified, fix being tested

---

## Problem

Windows tests in GitHub Actions were failing with:
```
Error: sshifu-trust binary not found at C:\npm\prefix\node_modules\sshifu-trust\bin\sshifu-trust.exe
```

---

## Solution Approach

Created a manual testing workflow to iterate quickly without polluting production releases:

### New Files Created

1. **`.github/workflows/test-windows-manual.yml`**
   - Manual trigger workflow (`workflow_dispatch`)
   - Builds binaries from any branch
   - Creates public prerelease (not draft)
   - Publishes to npm with `test` tag
   - Runs full test suite including Windows

2. **`scripts/test-windows-manual.sh`**
   - GitHub CLI helper for triggering and managing test runs
   - Commands: `trigger`, `status`, `logs`, `list`, `cleanup`

3. **Updated `.github/workflows/test-npm-release.yml`**
   - Added `workflow_call` trigger
   - Accepts `npm_tag` and `version` inputs for reusable testing

---

## Root Cause Discovery

### Issue 1: Draft Release Assets Not Accessible
- **Problem:** Draft releases don't expose assets for download
- **Fix:** Changed `draft: true` to `draft: false, prerelease: true`

### Issue 2: Input Name Mismatch
- **Problem:** `create-release` vs `create_release` (hyphen vs underscore)
- **Fix:** Corrected to `github.event.inputs.create_release`

### Issue 3: File Lock on Windows (CRITICAL)
- **Problem:** npm holds a lock on the downloaded `.zip` file, preventing PowerShell's `Expand-Archive` from accessing it
- **Error:** `The process cannot access the file because it is being used by another process`
- **Attempts:**
  1. Added 500ms delay - ❌ Failed
  2. Added retry logic with exponential backoff (5 retries) - ❌ Failed (file stays locked)
  3. Extract to temp directory - ❌ Failed (still locked)
  4. **Copy zip to temp file first, then extract** - ✅ Testing

---

## Install Script Evolution

### Final Fix (test.11+)

```javascript
if (isWindows) {
  // Copy zip to temp location first to avoid file lock issues
  const tempZip = path.join(binDir, 'temp_' + Date.now() + '.zip');
  fs.copyFileSync(archivePath, tempZip);
  
  const tempDir = path.join(binDir, 'tmp_extract_' + Date.now());
  fs.mkdirSync(tempDir, { recursive: true });
  
  try {
    // Extract from the COPY, not the original
    const psCommand = `Expand-Archive -Path '${tempZip}' -DestinationPath '${tempDir}' -Force -ErrorAction Stop`;
    execSync(`powershell -Command "${psCommand}"`, { stdio: ['ignore', 'pipe', 'pipe'] });
    
    // Find and move binary
    const extractedBin = tempFiles.find(f => f.endsWith('.exe') || !f.includes('.'));
    fs.renameSync(path.join(tempDir, extractedBin), binPath);
    
    // Cleanup all temp files
    fs.unlinkSync(tempZip);
    fs.rmSync(tempDir, { recursive: true, force: true });
    fs.unlinkSync(archivePath);
  } catch (extractErr) {
    // Cleanup on error
    try { fs.unlinkSync(tempZip); } catch (e) {}
    try { fs.rmSync(tempDir, { recursive: true, force: true }); } catch (e) {}
    throw extractErr;
  }
}
```

---

## Test Results

| Version | Changes | Windows Result |
|---------|---------|----------------|
| test.1 | Initial workflow | ❌ Draft release (404) |
| test.2 | Public prerelease | ❌ Input name mismatch |
| test.3 | Fixed input names | ❌ Postinstall script failed (silent) |
| test.4 | Added logging | ❌ File lock error revealed |
| test.5 | 500ms delay | ❌ Still locked |
| test.6 | Retry logic (5x) | ❌ File stays locked |
| test.7 | Better logging | ❌ All retries failed |
| test.8 | Temp dir extract | ❌ Still locked |
| test.9 | Longer retries | ❌ File permanently locked |
| test.10 | Temp dir only | ❌ Locked on extract |
| test.11 | **Copy zip first** | ⏳ Testing |

---

## Files Modified

### Install Scripts
- `packages/sshifu/scripts/install.js`
- `packages/sshifu-server/scripts/install.js`
- `packages/sshifu-trust/scripts/install.js`

### Workflows
- `.github/workflows/test-windows-manual.yml` (new)
- `.github/workflows/test-npm-release.yml` (updated)

### Scripts
- `scripts/test-windows-manual.sh` (new)

---

## Key Learnings

1. **npm File Locking on Windows**: npm holds exclusive locks on downloaded files during postinstall, preventing modification/extraction

2. **Copy-on-Write Workaround**: Copying the file creates a new file handle that isn't locked

3. **PowerShell Error Handling**: Use `-ErrorAction Stop` and capture stderr for proper error diagnosis

4. **Temp File Pattern**: Always clean up temp files in both success and error paths

5. **Testing Infrastructure**: Manual workflow + CLI helper enables rapid iteration

---

## Next Steps

1. **Verify test.11 passes** - Confirm copy-to-temp fix works
2. **Clean up test releases** - Delete v0.7.8-test.* releases from GitHub
3. **Merge to main** - Integrate fix into production
4. **Update documentation** - Document the Windows extraction workaround
5. **Consider improvements**:
   - Use native Node.js zip extraction (no PowerShell dependency)
   - Add Windows-specific test job to regular CI

---

## Usage

```bash
# Trigger new test
./scripts/test-windows-manual.sh trigger 0.7.8-test.12 fix/windows-npm-install-test

# Watch progress
gh run watch <run-id> --repo azophy/sshifu

# Cleanup after testing
./scripts/test-windows-manual.sh cleanup test
```

---

## Related Files

- Original issue: `dev-note-windows-error.md`
- Install scripts: `packages/*/scripts/install.js`
- Build script: `scripts/ci-build.sh`
- Release workflow: `.github/workflows/release.yml`

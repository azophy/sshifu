#!/usr/bin/env node

/**
 * Postinstall script for sshifu-trust
 * Downloads the appropriate binary from GitHub releases
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const PACKAGE_NAME = 'sshifu-trust';
const REPO = 'azophy/sshifu';

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;
  
  if (platform === 'linux') {
    if (arch === 'x64') return 'linux-amd64';
    if (arch === 'arm64') return 'linux-arm64';
    if (arch === 'arm') return 'linux-arm';
  }
  if (platform === 'darwin') {
    if (arch === 'x64') return 'darwin-amd64';
    if (arch === 'arm64') return 'darwin-arm64';
  }
  if (platform === 'win32') {
    if (arch === 'x64') return 'windows-amd64';
  }
  
  throw new Error(`Unsupported platform: ${platform} ${arch}`);
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        download(response.headers.location, dest).then(resolve).catch(reject);
        return;
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Download failed with status ${response.statusCode}`));
        return;
      }
      
      response.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', reject);
  });
}

async function main() {
  const platform = getPlatform();
  const binDir = path.join(__dirname, '..', 'bin');
  const binName = process.platform === 'win32' ? `${PACKAGE_NAME}.exe` : PACKAGE_NAME;
  const binPath = path.join(binDir, binName);
  
  // Ensure bin directory exists
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }
  
  // Get version from package.json
  const pkgPath = path.join(__dirname, '..', 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
  let version = pkg.version;
  
  // For dev versions, skip download (use local build)
  if (version.includes('dev')) {
    console.log(`[sshifu-trust] Dev version detected, skipping binary download`);
    console.log(`[sshifu-trust] Build manually with: go build -o bin/${binName} ./cmd/${PACKAGE_NAME}`);
    return;
  }
  
  const isWindows = process.platform === 'win32';
  const archiveExt = isWindows ? '.zip' : '.tar.gz';
  const archiveName = `${PACKAGE_NAME}-${platform}${archiveExt}`;
  const archiveUrl = `https://github.com/${REPO}/releases/download/v${version}/${archiveName}`;
  const archivePath = path.join(binDir, archiveName);

  console.log(`[sshifu-trust] Downloading ${archiveUrl}...`);

  try {
    await download(archiveUrl, archivePath);

    // Extract the archive
    console.log(`[sshifu-trust] Extracting...`);
    const archiveBinName = `${PACKAGE_NAME}-${platform}${isWindows ? '.exe' : ''}`;
    const extractedPath = path.join(binDir, archiveBinName);
    if (isWindows) {
      execSync(`powershell -Command "Expand-Archive -Path '${archivePath.replace(/'/g, "''")}' -DestinationPath '${binDir.replace(/'/g, "''")}' -Force"`, { stdio: 'ignore' });
    } else {
      execSync(`tar -xzf "${archivePath}" -C "${binDir}"`, { stdio: 'ignore' });
    }

    // Rename to expected name if different
    console.log(`[sshifu-trust] Looking for binary at: ${extractedPath}`);
    console.log(`[sshifu-trust] Target path: ${binPath}`);
    if (archiveBinName !== binName) {
      if (fs.existsSync(extractedPath)) {
        console.log(`[sshifu-trust] Renaming ${archiveBinName} to ${binName}`);
        fs.renameSync(extractedPath, binPath);
      } else {
        // Try to find the extracted file (in case of path issues)
        const files = fs.readdirSync(binDir);
        console.log(`[sshifu-trust] Files in binDir: ${files.join(', ')}`);
        const exeFile = files.find(f => f.endsWith('.exe') && f.startsWith(PACKAGE_NAME));
        if (exeFile) {
          console.log(`[sshifu-trust] Found ${exeFile}, renaming to ${binName}`);
          fs.renameSync(path.join(binDir, exeFile), binPath);
        }
      }
    }

    // Make executable on Unix
    if (process.platform !== 'win32') {
      fs.chmodSync(binPath, 0o755);
    }

    // Clean up archive
    fs.unlinkSync(archivePath);

    console.log(`[sshifu-trust] Binary installed successfully!`);
  } catch (err) {
    console.error(`[sshifu-trust] Installation failed: ${err.message}`);
    console.error(`[sshifu-trust] You can download the binary manually from https://github.com/${REPO}/releases`);
    process.exit(1);
  }
}

main();

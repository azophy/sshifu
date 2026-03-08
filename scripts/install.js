#!/usr/bin/env node

/**
 * Postinstall script for sshifu
 * Downloads the appropriate binary for the current platform from GitHub Releases
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const zlib = require('zlib');
const { execSync } = require('child_process');

const PACKAGE_NAME = 'sshifu';
const BINARIES = ['sshifu', 'sshifu-server', 'sshifu-trust'];

// Map npm platform/arch to Go GOOS/GOARCH
const PLATFORM_MAP = {
  'linux-x64': { os: 'linux', arch: 'amd64' },
  'linux-arm64': { os: 'linux', arch: 'arm64' },
  'linux-arm': { os: 'linux', arch: 'arm' },
  'darwin-x64': { os: 'darwin', arch: 'amd64' },
  'darwin-arm64': { os: 'darwin', arch: 'arm64' },
  'win32-x64': { os: 'windows', arch: 'amd64' },
};

function getPlatformInfo() {
  const npmPlatform = `${process.platform}-${process.arch}`;
  const mapped = PLATFORM_MAP[npmPlatform];
  
  if (!mapped) {
    throw new Error(
      `Unsupported platform: ${npmPlatform}\n` +
      `Supported platforms: ${Object.keys(PLATFORM_MAP).join(', ')}`
    );
  }
  
  return mapped;
}

function getBinaryExtension() {
  return process.platform === 'win32' ? '.exe' : '';
}

function getArchiveExtension() {
  return process.platform === 'win32' ? '.zip' : '.tar.gz';
}

function getDownloadUrl(version, platformInfo) {
  const ext = getArchiveExtension();
  const archiveName = `${PACKAGE_NAME}-${platformInfo.os}-${platformInfo.arch}${ext}`;
  return `https://github.com/azophy/sshifu/releases/download/${version}/${archiveName}`;
}

function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: '/repos/azophy/sshifu/releases/latest',
      method: 'GET',
      headers: {
        'User-Agent': PACKAGE_NAME,
        'Accept': 'application/vnd.github.v3+json',
      },
    };

    https.get(options, (res) => {
      let data = '';
      
      res.on('data', (chunk) => {
        data += chunk;
      });
      
      res.on('end', () => {
        if (res.statusCode === 200) {
          try {
            const parsed = JSON.parse(data);
            resolve(parsed.tag_name || parsed.name);
          } catch (e) {
            reject(new Error('Failed to parse GitHub API response'));
          }
        } else {
          reject(new Error(`GitHub API returned status ${res.statusCode}`));
        }
      });
    }).on('error', reject);
  });
}

function downloadFile(url) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        // Follow redirect
        downloadFile(res.headers.location).then(resolve).catch(reject);
        return;
      }
      
      if (res.statusCode !== 200) {
        reject(new Error(`Download failed with status ${res.statusCode}`));
        return;
      }
      
      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => resolve(Buffer.concat(chunks)));
    }).on('error', reject);
  });
}

function extractArchive(archiveData, destDir) {
  const ext = getArchiveExtension();
  
  if (ext === '.tar.gz') {
    return new Promise((resolve, reject) => {
      const tar = require('child_process').spawn('tar', ['xzf', '-', '-C', destDir], {
        stdio: ['pipe', 'inherit', 'inherit'],
      });
      
      tar.on('close', (code) => {
        if (code === 0) resolve();
        else reject(new Error(`tar exited with code ${code}`));
      });
      
      tar.stdin.write(archiveData);
      tar.stdin.end();
    });
  } else if (ext === '.zip') {
    // For Windows, write to temp file and extract
    const tempFile = path.join(destDir, 'temp.zip');
    fs.writeFileSync(tempFile, archiveData);
    
    try {
      // Try PowerShell first
      execSync(`powershell -command "Expand-Archive -Path '${tempFile}' -DestinationPath '${destDir}' -Force"`, {
        stdio: 'inherit',
      });
      fs.unlinkSync(tempFile);
      return Promise.resolve();
    } catch (e) {
      // Fallback to other methods if needed
      fs.unlinkSync(tempFile);
      return Promise.reject(new Error('Failed to extract zip archive'));
    }
  }
  
  return Promise.reject(new Error(`Unsupported archive format: ${ext}`));
}

function makeExecutable(filePath) {
  if (process.platform !== 'win32') {
    fs.chmodSync(filePath, 0o755);
  }
}

async function install() {
  console.log(`🔐 ${PACKAGE_NAME} installer`);
  console.log('========================\n');
  
  const platformInfo = getPlatformInfo();
  const binExt = getBinaryExtension();
  
  console.log(`Platform: ${process.platform}-${process.arch}`);
  console.log(`Mapped to: ${platformInfo.os}-${platformInfo.arch}\n`);
  
  // Determine version
  const version = process.env.SSHIFU_VERSION || await getLatestVersion();
  console.log(`Version: ${version}`);
  
  // Get download URL
  const downloadUrl = getDownloadUrl(version, platformInfo);
  console.log(`Downloading from: ${downloadUrl}\n`);
  
  // Download archive
  console.log('Downloading archive...');
  let archiveData;
  try {
    archiveData = await downloadFile(downloadUrl);
  } catch (e) {
    console.error(`Failed to download: ${e.message}`);
    console.error('\nMake sure you have a stable internet connection.');
    console.error('If the problem persists, download manually from:');
    console.error('https://github.com/azophy/sshifu/releases\n');
    process.exit(1);
  }
  
  // Create bin directory
  const binDir = path.join(__dirname, '..', 'bin');
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }
  
  // Create temp directory for extraction
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), `${PACKAGE_NAME}-`));
  
  try {
    // Extract archive
    console.log('Extracting binaries...');
    await extractArchive(archiveData, tempDir);
    
    // Move binaries to bin directory
    console.log('Installing binaries...');
    for (const binary of BINARIES) {
      const srcName = binary + binExt;
      const srcPath = path.join(tempDir, srcName);
      const destPath = path.join(binDir, srcName);
      
      if (fs.existsSync(srcPath)) {
        fs.copyFileSync(srcPath, destPath);
        makeExecutable(destPath);
        console.log(`  ✓ ${binary}${binExt}`);
      } else {
        console.warn(`  ⚠ ${binary}${binExt} not found in archive`);
      }
    }
    
    console.log('\n✅ Installation complete!');
    console.log(`\nYou can now run:`);
    console.log(`  ${BINARIES.map(b => b + binExt).join(', ')}`);
    console.log(`\nOr use npx:`);
    console.log(`  npx sshifu <server> <target>`);
    
  } catch (e) {
    console.error(`Installation failed: ${e.message}`);
    process.exit(1);
  } finally {
    // Cleanup temp directory
    try {
      fs.rmSync(tempDir, { recursive: true, force: true });
    } catch (e) {
      // Ignore cleanup errors
    }
  }
}

install();

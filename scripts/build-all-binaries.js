#!/usr/bin/env node

/**
 * Build all binaries for all platforms
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const BINARIES = ['sshifu', 'sshifu-server', 'sshifu-trust'];
const PLATFORMS = [
  { os: 'linux', arch: 'amd64' },
  { os: 'linux', arch: 'arm64' },
  { os: 'linux', arch: 'arm' },
  { os: 'darwin', arch: 'amd64' },
  { os: 'darwin', arch: 'arm64' },
  { os: 'windows', arch: 'amd64' },
];

function main() {
  const version = process.argv[2] || 'dev';
  const distDir = path.join(__dirname, '..', 'dist');
  
  if (!fs.existsSync(distDir)) {
    fs.mkdirSync(distDir, { recursive: true });
  }
  
  console.log(`Building binaries for version: ${version}`);
  
  for (const binary of BINARIES) {
    console.log(`\nBuilding ${binary}...`);
    
    for (const { os, arch } of PLATFORMS) {
      const ext = os === 'windows' ? '.exe' : '';
      const outputName = `${binary}-${os}-${arch}${ext}`;
      const outputPath = path.join(distDir, outputName);
      
      const cmd = `GOOS=${os} GOARCH=${arch} go build -o "${outputPath}" ./cmd/${binary}`;
      console.log(`  Building ${outputName}...`);
      
      try {
        execSync(cmd, { stdio: 'inherit' });
        
        // Create tar.gz for this binary
        if (os !== 'windows') {
          const tarName = `${binary}-${os}-${arch}.tar.gz`;
          execSync(`tar -czf "${path.join(distDir, tarName)}" -C "${distDir}" "${outputName}"`);
          console.log(`  Created ${tarName}`);
        } else {
          // For Windows, create zip
          const zipName = `${binary}-${os}-${arch}.zip`;
          execSync(`zip -j "${path.join(distDir, zipName)}" "${outputPath}"`);
          console.log(`  Created ${zipName}`);
        }
        
        // Clean up the individual binary file
        fs.unlinkSync(outputPath);
      } catch (err) {
        console.error(`  Failed to build ${outputName}: ${err.message}`);
      }
    }
  }
  
  console.log('\nBuild complete! Artifacts in dist/');
}

main();

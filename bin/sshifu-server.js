#!/usr/bin/env node

/**
 * sshifu-server CLI wrapper
 * Executes the sshifu-server binary from the bin directory
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const binName = process.platform === 'win32' ? 'sshifu-server.exe' : 'sshifu-server';
const binPath = path.join(__dirname, binName);

if (!fs.existsSync(binPath)) {
  console.error(`Error: sshifu-server binary not found at ${binPath}`);
  console.error('The postinstall script may have failed. Try running:');
  console.error('  npm rebuild sshifu');
  process.exit(1);
}

const child = spawn(binPath, process.argv.slice(2), {
  stdio: 'inherit',
});

child.on('error', (err) => {
  console.error(`Failed to start sshifu-server: ${err.message}`);
  process.exit(1);
});

child.on('close', (code) => {
  process.exit(code);
});

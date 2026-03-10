#!/usr/bin/env node

/**
 * sshifu CLI wrapper
 * Executes the sshifu binary from the bin directory
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const binName = process.platform === 'win32' ? 'sshifu.exe' : 'sshifu';
const binPath = path.join(__dirname, binName);

if (!fs.existsSync(binPath)) {
  console.error(`Error: sshifu binary not found at ${binPath}`);
  console.error('The postinstall script may have failed. Try running:');
  console.error('  npm rebuild sshifu');
  process.exit(1);
}

const child = spawn(binPath, process.argv.slice(2), {
  stdio: 'inherit',
});

child.on('error', (err) => {
  console.error(`Failed to start sshifu: ${err.message}`);
  process.exit(1);
});

child.on('close', (code) => {
  process.exit(code);
});

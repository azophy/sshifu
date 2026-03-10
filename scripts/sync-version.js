#!/usr/bin/env node

/**
 * Version sync script
 * Updates all package versions in the workspace to match
 */

const fs = require('fs');
const path = require('path');

function main() {
  const args = process.argv.slice(2);
  const newVersion = args[0];
  
  if (!newVersion) {
    console.error('Usage: node scripts/sync-version.js <version>');
    console.error('Example: node scripts/sync-version.js 1.2.3');
    process.exit(1);
  }
  
  const packagesDir = path.join(__dirname, '..', 'packages');
  const packages = fs.readdirSync(packagesDir);
  
  console.log(`Syncing versions to ${newVersion}...`);
  
  for (const pkgName of packages) {
    const pkgPath = path.join(packagesDir, pkgName, 'package.json');
    if (!fs.existsSync(pkgPath)) continue;
    
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
    const oldVersion = pkg.version;
    pkg.version = newVersion;
    
    fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
    console.log(`  ${pkgName}: ${oldVersion} → ${newVersion}`);
  }
  
  // Also update root package.json
  const rootPkgPath = path.join(__dirname, '..', 'package.json');
  const rootPkg = JSON.parse(fs.readFileSync(rootPkgPath, 'utf8'));
  rootPkg.version = newVersion;
  fs.writeFileSync(rootPkgPath, JSON.stringify(rootPkg, null, 2) + '\n');
  
  console.log('Version sync complete!');
}

main();

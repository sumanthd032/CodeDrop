#!/usr/bin/env node

'use strict';

const { execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const isWindows = process.platform === 'win32';
const binPath = path.join(__dirname, 'bin', isWindows ? 'codedrop.exe' : 'codedrop');

if (!fs.existsSync(binPath)) {
  console.error('codedrop binary not found. Try reinstalling: npm install -g codedrop');
  process.exit(1);
}

try {
  execFileSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
} catch (err) {
  process.exit(err.status || 1);
}

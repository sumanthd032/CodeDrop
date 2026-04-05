#!/usr/bin/env node

'use strict';

const https = require('https');
const fs = require('fs');
const path = require('path');

const pkg = require('./package.json');
const version = pkg.version;

const PLATFORM_MAP = {
  linux: 'linux',
  darwin: 'darwin',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];

if (!platform) {
  console.error(`Unsupported platform: ${process.platform}`);
  console.error('Supported platforms: linux, darwin, win32');
  process.exit(1);
}

if (!arch) {
  console.error(`Unsupported architecture: ${process.arch}`);
  console.error('Supported architectures: x64, arm64');
  process.exit(1);
}

const isWindows = process.platform === 'win32';
const binaryName = `codedrop-${platform}-${arch}${isWindows ? '.exe' : ''}`;
const downloadURL = `https://github.com/sumanthd032/codedrop/releases/download/v${version}/${binaryName}`;

const binDir = path.join(__dirname, 'bin');
const binPath = path.join(binDir, isWindows ? 'codedrop.exe' : 'codedrop');

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log(`Installing codedrop v${version} (${platform}/${arch})...`);

function download(url, dest, redirectsLeft) {
  return new Promise((resolve, reject) => {
    if (redirectsLeft === 0) {
      return reject(new Error('Too many redirects while downloading binary'));
    }

    const file = fs.createWriteStream(dest);

    https.get(url, { headers: { 'User-Agent': 'codedrop-npm-installer' } }, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        file.close();
        try { fs.unlinkSync(dest); } catch (_) {}
        return resolve(download(res.headers.location, dest, redirectsLeft - 1));
      }

      if (res.statusCode !== 200) {
        file.close();
        try { fs.unlinkSync(dest); } catch (_) {}
        return reject(new Error(
          `Download failed: HTTP ${res.statusCode}\n` +
          `URL: ${url}\n` +
          `Make sure release v${version} exists at https://github.com/sumanthd032/codedrop/releases`
        ));
      }

      res.pipe(file);
      file.on('finish', () => file.close(resolve));
      file.on('error', (err) => {
        try { fs.unlinkSync(dest); } catch (_) {}
        reject(err);
      });
    }).on('error', (err) => {
      try { fs.unlinkSync(dest); } catch (_) {}
      reject(err);
    });
  });
}

download(downloadURL, binPath, 5)
  .then(() => {
    if (!isWindows) {
      fs.chmodSync(binPath, '755');
    }
    console.log(`codedrop installed to ${binPath}`);
    console.log('\nQuick start:');
    console.log('  export CODEDROP_SERVER=https://your-api.example.com');
    console.log('  codedrop push <file>');
    console.log('  codedrop pull <url>');
  })
  .catch((err) => {
    console.error(`\nFailed to install codedrop: ${err.message}`);
    console.error('\nAlternatively, download manually from:');
    console.error(`  https://github.com/sumanthd032/codedrop/releases/download/v${version}/${binaryName}`);
    process.exit(1);
  });

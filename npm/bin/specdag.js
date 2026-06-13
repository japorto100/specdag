#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");

const platform = `${process.platform}-${process.arch}`;
const binName = process.platform === "win32" ? "specdag.exe" : "specdag";

// Map to vendor binary name
const VENDOR_MAP = {
  "linux-x64":   "specdag-linux-x64",
  "linux-arm64": "specdag-linux-arm64",
  "darwin-x64":  "specdag-darwin-x64",
  "darwin-arm64": "specdag-darwin-arm64",
  "win32-x64":   "specdag-win-x64.exe",
  "win32-arm64": "specdag-win-arm64.exe",
};

const vendorBin = VENDOR_MAP[platform];
if (!vendorBin) {
  console.error(`specdag: unsupported platform ${platform}`);
  process.exit(1);
}

// Find binary: follow symlinks to get real package location
const wrapperDir = path.dirname(fs.realpathSync(__filename));
const packageRoot = path.resolve(wrapperDir, "..");

const candidates = [
  path.join(packageRoot, "vendor", vendorBin),
  path.join(packageRoot, vendorBin),
];

let binPath = null;
for (const c of candidates) {
  if (fs.existsSync(c)) {
    binPath = c;
    break;
  }
}

if (!binPath) {
  console.error(`specdag: binary not found for ${platform}`);
  console.error(`Expected: vendor/${vendorBin}`);
  console.error(`Try: npm install -g @japorto100/specdag`);
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), {
  stdio: "inherit",
});

process.exit(result.status ?? 1);

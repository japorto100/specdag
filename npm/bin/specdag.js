#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");

const exe = process.platform === "win32" ? "specdag.exe" : "specdag";
const binPath = path.join(__dirname, "..", "vendor", exe);

if (!fs.existsSync(binPath)) {
  console.error("specdag binary not found. Try reinstalling the package.");
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), {
  stdio: "inherit"
});

process.exit(result.status ?? 1);

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const https = require("node:https");
const readline = require("node:readline");
const childProcess = require("node:child_process");

const pkg = require("../package.json");

const repo = "japorto100/specdag";
const version = `v${pkg.version}`;

function platformName() {
  switch (process.platform) {
    case "linux": return "Linux";
    case "darwin": return "Darwin";
    case "win32": return "Windows";
    default: throw new Error(`Unsupported platform: ${process.platform}`);
  }
}

function archName() {
  switch (process.arch) {
    case "x64": return "x86_64";
    case "arm64": return "arm64";
    default: throw new Error(`Unsupported architecture: ${process.arch}`);
  }
}

const platform = platformName();
const arch = archName();
const isWindows = process.platform === "win32";

const archiveExt = isWindows ? "zip" : "tar.gz";
const archiveName = `specdag_${platform}_${arch}.${archiveExt}`;
const url = `https://github.com/${repo}/releases/download/${version}/${archiveName}`;

const vendorDir = path.join(__dirname, "..", "vendor");
fs.mkdirSync(vendorDir, { recursive: true });

const archivePath = path.join(vendorDir, archiveName);

function download(fileUrl, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(fileUrl, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return download(res.headers.location, dest).then(resolve, reject);
      }

      if (res.statusCode !== 200) {
        reject(new Error(`Download failed: ${res.statusCode} ${res.statusMessage}`));
        return;
      }

      res.pipe(file);
      file.on("finish", () => {
        file.close(resolve);
      });
    }).on("error", reject);
  });
}

function ask(question) {
  const rl = readline.createInterface({ input: process.stdin, output: process.stderr });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

(async () => {
  // Step 1: Download binary
  console.log(`Downloading ${url}`);
  await download(url, archivePath);

  if (isWindows) {
    childProcess.execFileSync("powershell.exe", [
      "-NoProfile",
      "-Command",
      `Expand-Archive -Force '${archivePath}' '${vendorDir}'`
    ], { stdio: "inherit" });
  } else {
    childProcess.execFileSync("tar", ["-xzf", archivePath, "-C", vendorDir], {
      stdio: "inherit"
    });
    fs.chmodSync(path.join(vendorDir, "specdag"), 0o755);
  }

  // clean up downloaded archive
  fs.unlinkSync(archivePath);

  console.log("\n✅ specdag binary installed.");

  // Step 2: Offer MCP setup
  const binPath = path.join(vendorDir, isWindows ? "specdag.exe" : "specdag");

  if (process.stdout.isTTY || process.stderr.isTTY) {
    const answer = await ask("\n🔧 Register specdag as MCP server? [y/N] ");
    if (answer.toLowerCase() === "y" || answer.toLowerCase() === "yes") {
      try {
        childProcess.execFileSync(binPath, ["mcp", "setup"], {
          stdio: "inherit",
          cwd: process.cwd(),
        });
      } catch {
        console.log("MCP setup skipped (run 'specdag mcp setup' later).");
      }
    } else {
      console.log("Skipped MCP setup. Run 'specdag mcp setup' when ready.");
    }
  } else {
    console.log("Non-interactive. Run 'specdag mcp setup' to register MCP server.");
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});

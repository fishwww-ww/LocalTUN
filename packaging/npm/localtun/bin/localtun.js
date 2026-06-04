#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const executableName = process.platform === "win32" ? "localtun.exe" : "localtun";
const executablePath = path.join(__dirname, "..", "vendor", executableName);

if (!fs.existsSync(executablePath)) {
  console.error("localtun binary not found. Try reinstalling @fishwww-ww/localtun.");
  process.exit(1);
}

const result = spawnSync(executablePath, process.argv.slice(2), {
  stdio: "inherit"
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);

const fs = require("node:fs");
const path = require("node:path");

const packageMap = {
  "darwin-arm64": "@fishwww-ww/localtun-darwin-arm64",
  "darwin-x64": "@fishwww-ww/localtun-darwin-x64",
  "linux-arm64": "@fishwww-ww/localtun-linux-arm64",
  "linux-x64": "@fishwww-ww/localtun-linux-x64",
  "win32-x64": "@fishwww-ww/localtun-win32-x64"
};

const platformKey = `${process.platform}-${process.arch}`;
const packageName = packageMap[platformKey];

if (!packageName) {
  console.error(`Unsupported platform: ${platformKey}`);
  process.exit(1);
}

let packageJsonPath;
try {
  packageJsonPath = require.resolve(`${packageName}/package.json`);
} catch (error) {
  console.error(`Missing optional package: ${packageName}`);
  console.error("Reinstall after publishing the matching platform package.");
  process.exit(1);
}

const packageDir = path.dirname(packageJsonPath);
const exeName = process.platform === "win32" ? "localtun.exe" : "localtun";
const sourcePath = path.join(packageDir, "bin", exeName);
const vendorDir = path.join(__dirname, "vendor");
const targetPath = path.join(vendorDir, exeName);

if (!fs.existsSync(sourcePath)) {
  console.error(`Binary file not found: ${sourcePath}`);
  process.exit(1);
}

fs.mkdirSync(vendorDir, { recursive: true });
fs.copyFileSync(sourcePath, targetPath);

if (process.platform !== "win32") {
  fs.chmodSync(targetPath, 0o755);
}

console.log(`Installed ${packageName} -> ${targetPath}`);

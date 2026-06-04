import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const version = process.argv[2];

if (!version) {
  console.error("Usage: node scripts/set-npm-version.mjs <version>");
  process.exit(1);
}

const packageDirs = [
  "packaging/npm/localtun",
  "packaging/npm/localtun-darwin-arm64",
  "packaging/npm/localtun-darwin-x64",
  "packaging/npm/localtun-linux-arm64",
  "packaging/npm/localtun-linux-x64",
  "packaging/npm/localtun-win32-x64"
];

for (const dir of packageDirs) {
  const packageJsonPath = path.resolve(dir, "package.json");
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));

  packageJson.version = version;

  if (packageJson.optionalDependencies) {
    for (const dependencyName of Object.keys(packageJson.optionalDependencies)) {
      packageJson.optionalDependencies[dependencyName] = version;
    }
  }

  fs.writeFileSync(packageJsonPath, `${JSON.stringify(packageJson, null, 2)}\n`);
  console.log(`Updated ${path.relative(process.cwd(), packageJsonPath)} -> ${version}`);
}

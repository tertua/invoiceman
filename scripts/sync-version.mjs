// Sync web/package.json version from the root VERSION file.
// VERSION is the single source of truth; never bump package.json by hand.
// Usage: npm --prefix web run sync:version
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const versionPath = path.join(root, "VERSION");
const packagePath = path.join(root, "web", "package.json");

const version = (await readFile(versionPath, "utf8")).trim();
if (!/^\d+\.\d+\.\d+(-[\w.]+)?$/.test(version)) {
  throw new Error(`Invalid version in VERSION file: ${JSON.stringify(version)}`);
}

const pkg = JSON.parse(await readFile(packagePath, "utf8"));
if (pkg.version === version) {
  console.log(`web/package.json already at ${version}`);
  process.exit(0);
}
pkg.version = version;
await writeFile(packagePath, `${JSON.stringify(pkg, null, 2)}\n`);
console.log(`web/package.json bumped to ${version}`);

// Keep the release trio in lockstep from the root VERSION file:
//   1. web/package.json version == VERSION
//   2. web/CHANGELOG.md has a "## [vX.Y.Z]" section for VERSION
//   3. that section carries real notes (not the auto-inserted stub)
// VERSION is the single source of truth; never bump package.json by hand.
//
// Usage:
//   npm --prefix web run sync:version          # sync package.json + stub changelog
//   node ../scripts/sync-version.mjs --check   # CI: verify, never mutate
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const versionPath = path.join(root, "VERSION");
const packagePath = path.join(root, "web", "package.json");
const changelogPath = path.join(root, "web", "CHANGELOG.md");
const checkOnly = process.argv.includes("--check");

const STUB = "<!-- Describe user-visible changes: Added / Changed / Fixed / Security. -->";
const UNRELEASED = "## [Unreleased]";

const version = (await readFile(versionPath, "utf8")).trim();
if (!/^\d+\.\d+\.\d+(-[\w.]+)?$/.test(version)) {
  throw new Error(`Invalid version in VERSION file: ${JSON.stringify(version)}`);
}

const heading = `## [v${version}]`;

// --- web/package.json -------------------------------------------------------
const pkg = JSON.parse(await readFile(packagePath, "utf8"));
let pkgChanged = false;
if (pkg.version !== version) {
  if (checkOnly) {
    console.error(`VERSION (${version}) != web/package.json (${pkg.version}). Run: npm --prefix web run sync:version`);
    process.exitCode = 1;
  } else {
    pkg.version = version;
    await writeFile(packagePath, `${JSON.stringify(pkg, null, 2)}\n`);
    pkgChanged = true;
  }
}

// --- web/CHANGELOG.md -------------------------------------------------------
let changelog = await readFile(changelogPath, "utf8");

// Pull the body of the section that starts with `heading`, up to the next
// level-2 heading (or EOF).
function sectionBody(text) {
  const lines = text.split("\n");
  const start = lines.findIndex((line) => line.startsWith(heading));
  if (start === -1) return null;
  const rest = lines.slice(start + 1);
  const end = rest.findIndex((line) => line.startsWith("## "));
  return (end === -1 ? rest : rest.slice(0, end)).join("\n");
}

const hasHeading = changelog.includes(heading);
const body = hasHeading ? sectionBody(changelog) : null;
const hasNotes = body !== null && body.replace(STUB, "").trim() !== "";

if (checkOnly) {
  if (!hasHeading) {
    console.error(`web/CHANGELOG.md has no "${heading}" section. Run: npm --prefix web run sync:version`);
    process.exitCode = 1;
  } else if (!hasNotes) {
    console.error(`web/CHANGELOG.md "${heading}" is still the stub. Write user-visible notes (Added/Changed/Fixed/Security).`);
    process.exitCode = 1;
  }
} else {
  if (pkgChanged) console.log(`web/package.json bumped to ${version}`);
  if (!hasHeading) {
    const date = new Date().toISOString().slice(0, 10);
    const stub = `${heading} - ${date}\n\n${STUB}\n`;
    changelog = changelog.includes(UNRELEASED)
      ? changelog.replace(`${UNRELEASED}\n\n`, `${UNRELEASED}\n\n${stub}\n`)
      : `${changelog.trimEnd()}\n\n${stub}\n`;
    await writeFile(changelogPath, changelog);
    console.log(`web/CHANGELOG.md: added "${heading} - ${date}" stub — fill in user-visible notes`);
  } else if (!hasNotes) {
    console.log(`web/CHANGELOG.md "${heading}" still needs user-visible notes`);
  } else {
    console.log(`web/CHANGELOG.md "${heading}" section present`);
  }
}

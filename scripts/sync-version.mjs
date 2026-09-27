// Keep the release trio in lockstep from the root VERSION file:
//   1. web/package.json version == VERSION
//   2. web/CHANGELOG.md has a "## [vX.Y.Z]" section for VERSION
//   3. that section carries real notes (not the auto-inserted stub)
//   4. the bump itself is a single-component +1 release bump (guard below)
// VERSION is the single source of truth; never bump package.json by hand.
//
// Usage:
//   npm --prefix web run sync:version          # sync package.json + stub changelog
//   node ../scripts/sync-version.mjs --check   # CI: verify, never mutate
import { readFile, writeFile } from "node:fs/promises";
import { execSync } from "node:child_process";
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

// --- bump guard -------------------------------------------------------------
// A release bump may change exactly one component by +1, resetting every
// lower component to 0: 0.6.1 -> 0.6.2 (patch), 0.6.2 -> 0.7.0 (minor),
// 0.7.0 -> 1.0.0 (major). Jumps (0.6.1 -> 0.8.0), downgrades and skips are
// rejected so AI agents cannot run away with the version number. The
// reference is the last committed VERSION (uncommitted bump vs HEAD) or,
// when the worktree is clean, the previous commit that touched VERSION.
function triple(v) {
  const m = /^(\d+)\.(\d+)\.(\d+)/.exec(v);
  return m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null;
}

function bumpKind(prev, next) {
  const p = triple(prev);
  const n = triple(next);
  if (!p || !n) return null;
  if (p[0] === n[0] && p[1] === n[1] && p[2] === n[2]) return "same";
  if (n[0] === p[0] + 1 && n[1] === 0 && n[2] === 0) return "major";
  if (n[0] === p[0] && n[1] === p[1] + 1 && n[2] === 0) return "minor";
  if (n[0] === p[0] && n[1] === p[1] && n[2] === p[2] + 1) return "patch";
  return null;
}

const gitOut = (cmd) => execSync(cmd, { cwd: root, encoding: "utf8" }).trim();

try {
  let base;
  const head = gitOut("git show HEAD:VERSION");
  if (head !== version) {
    base = head;
  } else {
    const hashes = gitOut("git log -2 --format=%H -- VERSION").split("\n").filter(Boolean);
    if (hashes.length >= 2) base = gitOut(`git show ${hashes[1]}:VERSION`);
  }
  if (base !== undefined && bumpKind(base, version) === null) {
    console.error(
      `FAIL: VERSION ${base} -> ${version} is not a valid release bump. ` +
        `Allowed: one component +1 with lower ones reset (patch 0.6.1->0.6.2, minor 0.6.2->0.7.0, major 0.7.0->1.0.0).`
    );
    process.exit(1);
  }
} catch {
  // Not a git checkout (or VERSION is new): bump guard does not apply.
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

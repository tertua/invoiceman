// Guard against docs/MODULE_MAP.md going stale (both directions):
//   1. every owner file on disk must be referenced in the map (by relative
//      path or basename), so new domains can't slip in undocumented;
//   2. every backticked .go/.js/.jsx path in the map must exist, so renames
//      and deletions can't leave dead pointers.
// Usage: bun run --cwd=webui check:map
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const mapPath = path.join(root, "docs", "MODULE_MAP.md");

const map = await readFile(mapPath, "utf8");

async function listFiles(dir, exts) {
  const out = [];
  for (const e of await readdir(path.join(root, dir))) {
    if (e.endsWith("_test.go")) continue;
    if (exts.some((x) => e.endsWith(x))) out.push(path.posix.join(dir, e));
  }
  return out;
}

const owned = [
  ...(await listFiles("app/controllers", [".go"])),
  ...(await listFiles("app/models", [".go"])),
  ...(await listFiles("app/queries", [".go"])),
  ...(await listFiles("pkg/routes", [".go"])),
  ...(await listFiles("webui/src/api", [".js"])),
  ...(await listFiles("webui/src/hooks", [".js"])),
  ...(await listFiles("webui/src/pages", [".jsx"])),
  ...(await listFiles("webui/src/context", [".jsx"])),
];

// Top-level platform areas (files or packages, docs excluded).
for (const e of await readdir(path.join(root, "platform"))) {
  if (e.endsWith(".md")) continue;
  owned.push(path.posix.join("platform", e));
}

const missing = owned.filter((f) => {
  if (map.includes(f) || map.includes(path.posix.basename(f))) return false;
  // app/queries mirrors the domains one file each (controller/model/api
  // already force a map update for new domains); the layer itself is
  // documented once in the request-path section.
  if (path.posix.dirname(f) === "app/queries" && map.includes("app/queries")) {
    return false;
  }
  return true;
});

// Reverse: every backticked path in the map must resolve on disk — by exact
// path, or by basename anywhere in the repo (table cells often use basenames;
// generated/test files are matched this way too).
const basenames = new Set();
async function collectBasenames(dir) {
  for (const e of await readdir(path.join(root, dir), { withFileTypes: true })) {
    if (["node_modules", ".git", ".vscode-server", "dist"].includes(e.name)) {
      continue;
    }
    const full = path.join(root, dir, e.name);
    if (e.isDirectory()) await collectBasenames(path.posix.join(dir, e.name));
    else basenames.add(e.name);
  }
}
await collectBasenames(".");
const { existsSync } = await import("node:fs");
const dead = [...map.matchAll(/`([^`]+)`/g)]
  .map((m) => m[1])
  .filter((p) => /\.(go|js|jsx)$/.test(p) && !p.includes("*"))
  .filter((p) => !existsSync(path.join(root, p)) && !basenames.has(path.posix.basename(p)));
const uniqueDead = [...new Set(dead)];

let failed = false;
for (const f of missing) {
  console.error(`unmapped owner file: ${f}`);
  failed = true;
}
for (const f of uniqueDead) {
  console.error(`dead map reference: ${f}`);
  failed = true;
}
if (failed) {
  console.error("\nUpdate docs/MODULE_MAP.md, then re-run: bun run --cwd=webui check:map");
  process.exit(1);
}
console.log(`map OK: ${owned.length} owner files referenced, no dead links.`);

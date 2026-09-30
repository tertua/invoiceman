// Guard against thousand-line files creeping back in.
// Model: ratchet, not cliff. Baseline (file-size-baseline.json) locks every
// file at its line count on the day this check landed; a file may shrink but
// never grow. Brand-new files must fit the category BUDGETS below.
// Why not flat budgets? 15 legacy files predate the budgets — failing CI on
// day one would just train everyone to ignore the check. Splits lower the
// baseline entry; growth fails loudly.
//
// For AI assistants: one file = one responsibility. When a file approaches
// its budget, split it the way the repo already does: controllers by route
// area (see gateway_*_controller.go), pages by extracting components/*
// cards (see InvoiceDetail.jsx split), flow tests by domain (see
// flow_*_test.go), i18n by language (see i18n.en/id.js).
//
// Usage: bun run --cwd=webui check:size
// After an intentional split, lower that file's baseline entry (never raise).
import { execSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const baselinePath = path.join(import.meta.dirname, "file-size-baseline.json");

const BUDGETS = [
  ["app/controllers/", 400],
  ["app/queries/", 300],
  ["app/models/", 200],
  ["pkg/routes/", 400],
  ["webui/src/pages/", 250],
  ["webui/src/components/", 250],
  ["webui/src/lib/i18n.", 900], // per-language dictionaries
  ["webui/src/hooks/", 150],
  ["webui/src/api/", 150],
];
const DEFAULT_BUDGET = 400;
const GENERATED = new Set(["docs/docs.go"]);

function budgetFor(file) {
  for (const [prefix, budget] of BUDGETS) {
    if (file.startsWith(prefix)) return budget;
  }
  return DEFAULT_BUDGET;
}

const baseline = JSON.parse(await readFile(baselinePath, "utf8"));
const files = execSync("git ls-files", { encoding: "utf8", cwd: root })
  .split("\n")
  .filter((f) => /\.(go|jsx?)$/.test(f) && !f.includes("node_modules") && !GENERATED.has(f));

let failed = false;
for (const f of files) {
  let lines = 0;
  try {
    lines = Number(execSync(`wc -l < ${JSON.stringify(f)}`, { encoding: "utf8", shell: "/bin/bash", cwd: root }).trim());
  } catch {
    continue; // file removed mid-check; git status will explain
  }
  if (f in baseline) {
    if (lines > baseline[f]) {
      console.error(`grew: ${f} went ${baseline[f]} → ${lines} lines — split it, don't append`);
      failed = true;
    }
  } else if (lines > budgetFor(f)) {
    console.error(`oversize new file: ${f} has ${lines} lines (budget ${budgetFor(f)}) — split by structure, don't land it whole`);
    failed = true;
  }
}
if (failed) {
  console.error("\nSplit the file (see header comment), then re-run: bun run --cwd=webui check:size");
  process.exit(1);
}
console.log(`size OK: ${files.length} files within ratchet.`);

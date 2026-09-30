// Guard the money-as-string charting regression.
//
// Backend `models.Money` is a shopspring decimal, which JSON-encodes as a
// string. Recharts' Pie silently renders zero slices for string values because
// its slice total only accepts the number type. Every Pie must therefore pass
// its data through `chartNumbers(...)` (src/lib/chartData.js) — either inline
// in the tag or when the data variable is declared.
//
// Usage: bun run --cwd=webui check:charts
import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const sourceRoot = path.join(root, "src");
const COERCER = "chartNumbers";

async function jsxFiles(directory) {
  const out = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) out.push(...(await jsxFiles(file)));
    else if (entry.name.endsWith(".jsx")) out.push(file);
  }
  return out;
}

// A pie is safe when its data prop calls the coercer inline, or references a
// variable declared from it (`const data = chartNumbers(...)`).
function pieIsCoerced(tag, content) {
  if (tag.includes(`${COERCER}(`)) return true;
  const dataProp = tag.match(/data=\{([A-Za-z_$][\w$]*)\}/);
  if (!dataProp) return false;
  const name = dataProp[1];
  return new RegExp(`(?:const|let|var)\\s+${name}\\s*=\\s*${COERCER}\\(`).test(content);
}

const violations = [];
for (const file of await jsxFiles(sourceRoot)) {
  const relative = path.relative(sourceRoot, file).split(path.sep).join("/");
  const content = await readFile(file, "utf8");
  for (const match of content.matchAll(/<Pie\b[^>]*>/g)) {
    const tag = match[0];
    if (!tag.includes("dataKey=")) continue;
    if (pieIsCoerced(tag, content)) continue;
    violations.push(`${relative}: <Pie> data must go through ${COERCER}(...) so decimal-string money renders`);
  }
}

if (violations.length) {
  for (const message of violations) console.error(message);
  console.error(`\nWrap the pie data (see src/lib/chartData.js), then re-run: bun run --cwd=webui check:charts`);
  process.exit(1);
}
console.log("charts OK: every <Pie> coerces its data.");

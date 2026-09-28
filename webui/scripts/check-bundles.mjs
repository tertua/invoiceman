import { gzipSync } from "node:zlib";
import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const sourceRoot = path.join(root, "src");
const distRoot = path.join(root, "dist", "assets");
const baselinePath = path.join(root, "bundle-baseline.json");
const strict = process.env.BUNDLE_BUDGET === "strict";

const pdfImportAllowlist = new Set([
  "components/invoice/InvoiceDocument.jsx",
  "components/invoice/InvoicePdfDownloadContent.jsx",
]);
const chartImportAllowlist = new Set([
  "components/dashboard/DashboardCharts.jsx",
  "components/clients/ClientCharts.jsx",
  "components/reports/ReportsCharts.jsx",
]);

async function filesIn(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...(await filesIn(file)));
    else if (/\.(jsx?|tsx?)$/.test(entry.name)) files.push(file);
  }
  return files;
}

function relativeSource(file) {
  return path.relative(sourceRoot, file).split(path.sep).join("/");
}

async function checkImports() {
  const violations = [];
  for (const file of await filesIn(sourceRoot)) {
    const relative = relativeSource(file);
    const content = await readFile(file, "utf8");
    if (content.includes("@react-pdf/renderer") && !pdfImportAllowlist.has(relative)) {
      violations.push(`PDF import is outside the on-demand boundary: src/${relative}`);
    }
    if (/from\s+["']recharts["']/.test(content) && !chartImportAllowlist.has(relative)) {
      violations.push(`Recharts import is outside the chart pages: src/${relative}`);
    }
  }
  return violations;
}

async function bundleSizes() {
  const files = (await readdir(distRoot)).filter((file) => file.endsWith(".js"));
  return Promise.all(
    files.map(async (file) => {
      const content = await readFile(path.join(distRoot, file));
      return {
        file,
        bytes: content.length,
        gzipBytes: gzipSync(content, { level: 9 }).length,
      };
    })
  );
}

function formatBytes(bytes) {
  return `${(bytes / 1024).toFixed(1)} KB`;
}

// Resolve an entry chunk from its HTML file instead of a filename regex — dist holds other admin-*.js chunks (e.g. src/api/admin.js).
async function entryFile(htmlName) {
  try {
    const html = await readFile(path.join(root, "dist", htmlName), "utf8");
    const match = html.match(/<script[^>]+src="\/assets\/([\w-]+\.js)"/);
    return match ? match[1] : null;
  } catch {
    return null;
  }
}

async function main() {
  const baseline = JSON.parse(await readFile(baselinePath, "utf8"));
  const violations = await checkImports();
  let bundles;
  try {
    bundles = await bundleSizes();
  } catch {
    violations.push("dist/assets is missing; run npm run build first");
    bundles = [];
  }

  const entrySpecs = [
    { html: "index.html", label: "product entry", raw: baseline.entryRawBytes, gzip: baseline.entryGzipBytes },
    { html: "admin.html", label: "admin entry", raw: baseline.adminEntryRawBytes, gzip: baseline.adminEntryGzipBytes },
  ];

  const namedChunks = new Set(bundles.map(({ file }) => file.split("-")[0]));
  for (const required of baseline.requiredChunks) {
    if (!namedChunks.has(required)) violations.push(`${required} vendor chunk is missing`);
  }

  const budgetViolations = [];
  for (const spec of entrySpecs) {
    const file = await entryFile(spec.html);
    const entry = file && bundles.find((bundle) => bundle.file === file);
    if (!entry) {
      violations.push(`${spec.label} chunk missing (${spec.html} not in dist; run npm run build)`);
      continue;
    }
    if (entry.bytes > spec.raw) {
      budgetViolations.push(`${spec.label} raw size ${formatBytes(entry.bytes)} exceeds ${formatBytes(spec.raw)}`);
    }
    if (entry.gzipBytes > spec.gzip) {
      budgetViolations.push(`${spec.label} gzip size ${formatBytes(entry.gzipBytes)} exceeds ${formatBytes(spec.gzip)}`);
    }
  }

  console.log(`Bundle check (${strict ? "strict" : "advisory"})`);
  for (const bundle of bundles.sort((a, b) => b.bytes - a.bytes)) {
    console.log(`  ${bundle.file}: ${formatBytes(bundle.bytes)} raw, ${formatBytes(bundle.gzipBytes)} gzip`);
  }
  for (const message of [...violations, ...budgetViolations]) {
    console.warn(`${strict ? "ERROR" : "WARNING"}: ${message}`);
  }

  if (strict && (violations.length || budgetViolations.length)) process.exitCode = 1;
}

await main();

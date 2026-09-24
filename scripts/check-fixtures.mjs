// Guard against flow tests hand-writing invoice JSON again.
//
// The `invoiceSpec` builder in pkg/routes/flow_fixtures_test.go exists so a new
// cross-field rule (e.g. "sent requires a client") is added in one place and
// every test picks it up. Inline `POST /api/invoices` bodies drift: a test
// silently keeps building invalid payloads and only fails when the rule lands.
//
// This check fails when a *_test.go file (other than the fixture itself) posts
// a raw JSON literal to /api/invoices instead of using the builder.
//
// Usage: npm run check:fixtures (wired into CI + make web.check)
import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const routesDir = path.join(root, "pkg", "routes");
const FIXTURE = "flow_fixtures_test.go";

// A raw literal looks like: doRequest(t, app, "POST", "/api/invoices", `{
const INLINE = /doRequest\([^)]*"POST",\s*"\/api\/invoices",\s*`\{/s;

const files = (await readdir(routesDir)).filter((f) => f.endsWith("_test.go") && f !== FIXTURE);

const violations = [];
for (const file of files) {
  const content = await readFile(path.join(routesDir, file), "utf8");
  if (INLINE.test(content)) {
    violations.push(`pkg/routes/${file}: inline invoice JSON — use newInvoice()/createInvoice() from ${FIXTURE}`);
  }
}

if (violations.length) {
  for (const message of violations) console.error(message);
  console.error("\nBuild the payload with the invoiceSpec fixture, then re-run: npm run check:fixtures");
  process.exit(1);
}
console.log("fixtures OK: flow tests build invoices through the shared builder.");

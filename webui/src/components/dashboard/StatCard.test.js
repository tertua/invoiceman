import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

// node --test lacks Vite's resolution/JSX handling: register a resolver for
// "@/..." + extensionless imports and a loader that transpiles .jsx through
// bun's transpiler (same hook as OidcButton.test.js).
const base = new URL("../../", import.meta.url).href;
const nodeHook = `import { existsSync, readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
const base = ${JSON.stringify(base)};
export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith("@/")) {
    let rel = specifier.slice(2);
    if (!/\\.[a-z]+$/.test(rel)) {
      for (const ext of [".js", ".jsx"]) {
        const url = new URL(rel + ext, base).href;
        if (existsSync(fileURLToPath(url))) return nextResolve(url, context);
      }
      rel += ".js";
    }
    return nextResolve(new URL(rel, base).href, context);
  }
  if (/^\\.\\.?\\//.test(specifier) && !/\\.[a-z]+$/.test(specifier) && context.parentURL) {
    for (const ext of [".js", ".jsx"]) {
      const url = new URL(specifier + ext, context.parentURL).href;
      if (existsSync(fileURLToPath(url))) return nextResolve(url, context);
    }
  }
  return nextResolve(specifier, context);
}
export async function load(url, context, nextLoad) {
  if (url.endsWith(".jsx")) {
    const src = readFileSync(fileURLToPath(url), "utf8");
    const script = "const out = new Bun.Transpiler({loader:'jsx'}).transformSync(await new Response(Bun.stdin.stream()).text());" +
      "const names = [...new Set(out.match(/jsx[A-Za-z]*_[a-z0-9]+/g) || [])];" +
      "const imports = names.map((n) => 'import { jsxDEV as ' + n + ' } from \\"react/jsx-dev-runtime\\";').join('');" +
      "const frag = (out.match(/Fragment_[a-z0-9]+/g) || [])[0];" +
      "const fragImport = frag ? 'import { Fragment as ' + frag + ' } from \\"react\\";' : '';" +
      "process.stdout.write(fragImport + imports + out);";
    const out = execFileSync("bun", ["-e", script], { input: src, encoding: "utf8" });
    return { format: "module", source: out, shortCircuit: true };
  }
  return nextLoad(url, context);
}`;
register("data:text/javascript," + encodeURIComponent(nodeHook));

const { StatCard } = await import("./StatCard.jsx");
const { LangProvider } = await import("@/context/LangContext");

function render(props) {
  return renderToStaticMarkup(createElement(LangProvider, null, createElement(StatCard, props)));
}

// The settlement panel passes `sub` (e.g. "3 success") alongside the amount.
// StatCard used to drop it silently, so the count never reached the screen.
test("StatCard renders the sub count line", () => {
  const html = render({ label: "Settled total", value: "Rp 1.500.000", sub: "3 success" });
  assert.match(html, /3 success/);
  assert.match(html, /Settled total/);
});

test("StatCard renders no sub element when the prop is absent or empty", () => {
  const absent = render({ label: "Total expenses", value: "Rp 250.000" });
  const empty = render({ label: "Total expenses", value: "Rp 250.000", sub: "" });
  assert.equal(absent, empty);
});

test("StatCard renders a sparkline only when data is present", () => {
  const withData = render({ label: "Revenue", value: "1", data: [{ v: 1 }, { v: 4 }] });
  assert.match(withData, /<polyline/);

  const withoutData = render({ label: "Revenue", value: "1" });
  assert.doesNotMatch(withoutData, /<svg/);
});

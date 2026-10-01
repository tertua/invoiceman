import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

// node --test lacks Vite's resolution/JSX handling: register a resolver for
// "@/..." + extensionless imports, a loader that transpiles .jsx through bun's
// transpiler, and a stub for the config hook so the OIDC flag is controllable.
const base = new URL("../../", import.meta.url).href;
const configStub =
  "data:text/javascript," +
  encodeURIComponent("export const useOidcEnabled = () => globalThis.__oidcEnabled;");
const nodeHook = `import { existsSync, readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
const base = ${JSON.stringify(base)};
const configStub = ${JSON.stringify(configStub)};
export async function resolve(specifier, context, nextResolve) {
  if (specifier === "@/hooks/useConfig") return nextResolve(configStub, context);
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

const { OidcButton } = await import("./OidcButton.jsx");
const { LangProvider } = await import("@/context/LangContext");

function render(enabled) {
  globalThis.__oidcEnabled = enabled;
  return renderToStaticMarkup(createElement(LangProvider, null, createElement(OidcButton)));
}

test("OidcButton renders the SSO button and divider when enabled", () => {
  const html = render(true);
  assert.match(html, /Sign in with SSO/);
  assert.match(html, /or continue with/);
  assert.match(html, /\/api\/v1\/auth\/oidc\/login/);
});

test("OidcButton renders nothing when disabled", () => {
  assert.equal(render(false), "");
});

import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// node --test lacks Vite's resolution/JSX handling; see useInvoices.test.js.
const base = new URL("../", import.meta.url).href;
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
      "process.stdout.write(imports + out);";
    const out = execFileSync("bun", ["-e", script], { input: src, encoding: "utf8" });
    return { format: "module", source: out, shortCircuit: true };
  }
  return nextLoad(url, context);
}`;
register("data:text/javascript," + encodeURIComponent(nodeHook));

const { itemsKey } = await import("./useItems.js");
const { useItemMutations } = await import("./useItemMutations.js");
const { itemsApi } = await import("@/api/items");
const { LangProvider } = await import("@/context/LangContext");

function renderHook(client, useHook) {
  let captured;
  function Probe() {
    captured = useHook();
    return null;
  }
  renderToStaticMarkup(
    createElement(
      QueryClientProvider,
      { client },
      createElement(LangProvider, null, createElement(Probe)),
    ),
  );
  return captured;
}

function withStub(name, fn, run) {
  const original = itemsApi[name];
  itemsApi[name] = fn;
  return run().finally(() => {
    itemsApi[name] = original;
  });
}

test("useItemMutations.remove drops the row optimistically and restores on failure", async () => {
  const qc = new QueryClient();
  qc.setQueryData(itemsKey, [{ id: "a" }, { id: "b" }]);
  await withStub("remove", async () => {
    throw { status: 500 };
  }, async () => {
    const m = renderHook(qc, useItemMutations).remove;
    await assert.rejects(() => m.mutateAsync("a"));
    assert.deepEqual(qc.getQueryData(itemsKey).map((x) => x.id), ["a", "b"]);
  });
});

test("useItemMutations.create inserts a temp row, then swaps in the server entity", async () => {
  const qc = new QueryClient();
  qc.setQueryData(itemsKey, [{ id: "a", name: "A", rate: 1 }]);
  await withStub("create", async (payload) => ({ ...payload, id: "server-1" }), async () => {
    const m = renderHook(qc, useItemMutations).create;
    await m.mutateAsync({ name: "New", rate: 2 });
    const list = qc.getQueryData(itemsKey);
    assert.equal(list.length, 2);
    assert.equal(list[1].id, "server-1");
    assert.equal(list[1].name, "New");
  });
});

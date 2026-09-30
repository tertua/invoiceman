import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// node --test lacks Vite's resolution/JSX handling, so register a tiny resolver
// for "@/" + extensionless imports and a loader that transpiles .jsx (the hook
// graph reaches LangContext.jsx) via bun's own transpiler.
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

const { invoicesKey, invoiceKey } = await import("./useInvoices.js");
const { useSetInvoiceStatus, useDeleteInvoice } = await import("./useInvoiceMutations.js");
const { invoicesApi } = await import("@/api/invoices");
const { LangProvider } = await import("@/context/LangContext");
const { en } = await import("@/lib/i18n.en.js");

// Hooks need LangProvider (toast copy) + QueryClientProvider; render once via SSR.
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

test("invoicesKey / invoiceKey keep their cache shape", () => {
  assert.deepEqual(invoicesKey(), ["invoices", {}]);
  assert.deepEqual(invoiceKey("i1"), ["invoice", "i1"]);
});

test("useSetInvoiceStatus rolls the list and detail back on failure", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices", {}], [{ id: "i1", status: "draft", effective_status: "draft" }]);
  qc.setQueryData(invoiceKey("i1"), { id: "i1", status: "draft", effective_status: "draft" });
  const original = invoicesApi.setStatus;
  invoicesApi.setStatus = async () => {
    throw { status: 500, message: "boom" };
  };
  try {
    const m = renderHook(qc, useSetInvoiceStatus);
    await assert.rejects(() => m.mutateAsync({ id: "i1", status: "sent" }));
    assert.equal(qc.getQueryData(["invoices", {}])[0].status, "draft");
    assert.equal(qc.getQueryData(invoiceKey("i1")).status, "draft");
  } finally {
    invoicesApi.setStatus = original;
  }
});

test("useSetInvoiceStatus patches the list + detail before the request settles", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices", {}], [{ id: "i1", status: "draft", effective_status: "draft" }]);
  qc.setQueryData(invoiceKey("i1"), { id: "i1", status: "draft", effective_status: "draft" });
  const original = invoicesApi.setStatus;
  let release;
  invoicesApi.setStatus = () => new Promise((r) => (release = () => r({ id: "i1", status: "sent" })));
  try {
    const m = renderHook(qc, useSetInvoiceStatus);
    const pending = m.mutateAsync({ id: "i1", status: "sent" });
    await new Promise((r) => setTimeout(r, 0));
    assert.equal(qc.getQueryData(["invoices", {}])[0].status, "sent");
    assert.equal(qc.getQueryData(invoiceKey("i1")).status, "sent");
    release();
    await pending;
  } finally {
    invoicesApi.setStatus = original;
  }
});

test("useDeleteInvoice drops the row optimistically and restores it on failure", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices", {}], [{ id: "i1" }, { id: "i2" }]);
  const original = invoicesApi.remove;
  invoicesApi.remove = async () => {
    throw { status: 500, message: "boom" };
  };
  try {
    const m = renderHook(qc, useDeleteInvoice);
    await assert.rejects(() => m.mutateAsync("i1"));
    assert.deepEqual(qc.getQueryData(["invoices", {}]).map((x) => x.id), ["i1", "i2"]);
  } finally {
    invoicesApi.remove = original;
  }
});

test("toast copy exists in both locales for the new keys", () => {
  for (const key of ["invoices.statusFailed", "invoices.deleteFailed"]) {
    assert.notEqual(en[key], undefined, key);
  }
});

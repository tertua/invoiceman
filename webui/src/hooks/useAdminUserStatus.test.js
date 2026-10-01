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

const { adminUsersKey } = await import("./useAdminUsers.js");
const { useUpdateUserStatus } = await import("./useAdminUserStatus.js");
const { adminUsersApi } = await import("@/api/adminUsers");
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

test("useUpdateUserStatus calls the status endpoint and invalidates the users key", async () => {
  const qc = new QueryClient();
  qc.setQueryData(adminUsersKey, [{ id: "a", status: 1 }]);
  const original = adminUsersApi.updateUserStatus;
  let received;
  adminUsersApi.updateUserStatus = async (id, status) => {
    received = { id, status };
    return { id, status };
  };
  try {
    let invalidated = false;
    const spy = qc.invalidateQueries.bind(qc);
    qc.invalidateQueries = (args) => {
      invalidated = true;
      return spy(args);
    };
    const mutation = renderHook(qc, useUpdateUserStatus);
    await mutation.mutateAsync({ id: "a", status: 0 });
    assert.deepEqual(received, { id: "a", status: 0 });
    assert.equal(invalidated, true);
  } finally {
    adminUsersApi.updateUserStatus = original;
  }
});

test("useUpdateUserStatus rejects when the endpoint fails", async () => {
  const qc = new QueryClient();
  const original = adminUsersApi.updateUserStatus;
  adminUsersApi.updateUserStatus = async () => {
    throw { status: 500, message: "failed" };
  };
  try {
    const mutation = renderHook(qc, useUpdateUserStatus);
    await assert.rejects(() => mutation.mutateAsync({ id: "a", status: 0 }));
  } finally {
    adminUsersApi.updateUserStatus = original;
  }
});

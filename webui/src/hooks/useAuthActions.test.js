import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// node --test lacks Vite's resolution, so register a tiny resolver for "@/" and extensionless imports.
const resolveHook = `import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
const base = ${JSON.stringify(new URL("../", import.meta.url).href)};
export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith("@/")) {
    let rel = specifier.slice(2);
    if (!/\\.[a-z]+$/.test(rel)) rel += ".js";
    return nextResolve(new URL(rel, base).href, context);
  }
  if (/^\\.\\.?\\//.test(specifier) && !/\\.[a-z]+$/.test(specifier) && context.parentURL) {
    const url = new URL(specifier + ".js", context.parentURL).href;
    if (existsSync(fileURLToPath(url))) return nextResolve(url, context);
  }
  return nextResolve(specifier, context);
}`;
register("data:text/javascript," + encodeURIComponent(resolveHook));

const { useAuthActions } = await import("./useAuthActions.js");
const { authApi } = await import("@/api/auth");

// Hooks render once via SSR; effects (the 401 listener) need a DOM, actions do not.
function renderActions(client, setters) {
  let captured;
  function Probe() {
    captured = useAuthActions(setters);
    return null;
  }
  renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(Probe)));
  return captured;
}

function fakeSetters() {
  const calls = { user: [], loading: [], expired: [] };
  return {
    calls,
    setUser: (v) => calls.user.push(v),
    setLoading: (v) => calls.loading.push(v),
    setSessionExpired: (v) => calls.expired.push(v),
  };
}

async function withAuthStubs(stubs, run) {
  const original = {};
  for (const [name, fn] of Object.entries(stubs)) {
    original[name] = authApi[name];
    authApi[name] = fn;
  }
  try {
    return await run();
  } finally {
    Object.assign(authApi, original);
  }
}

test("useAuthActions returns the five auth actions", () => {
  const actions = renderActions(new QueryClient(), fakeSetters());
  for (const name of ["refresh", "login", "register", "logout", "updateProfile"]) {
    assert.equal(typeof actions[name], "function", name);
  }
});

test("refresh stores the session user and stops loading", async () => {
  const s = fakeSetters();
  const actions = renderActions(new QueryClient(), s);
  await withAuthStubs({ me: async () => ({ user: { id: "u1" } }) }, async () => {
    await actions.refresh();
  });
  assert.deepEqual(s.calls.user, [{ id: "u1" }]);
  assert.deepEqual(s.calls.expired, [false]);
  assert.deepEqual(s.calls.loading, [false]);
});

test("refresh drops the user when there is no session", async () => {
  const s = fakeSetters();
  const actions = renderActions(new QueryClient(), s);
  await withAuthStubs({ me: async () => { throw new Error("401"); } }, async () => {
    await actions.refresh();
  });
  assert.deepEqual(s.calls.user, [null]);
  assert.deepEqual(s.calls.loading, [false]);
});

test("login clears the previous account's cache before swapping identity", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices"], [{ id: "i1" }]);
  const s = fakeSetters();
  const actions = renderActions(qc, s);
  await withAuthStubs({ login: async () => ({ user: { id: "u2" } }) }, async () => {
    assert.deepEqual(await actions.login({ email: "a@b.c" }, "cap"), { id: "u2" });
  });
  assert.equal(qc.getQueryData(["invoices"]), undefined);
  assert.deepEqual(s.calls.user, [{ id: "u2" }]);
  assert.deepEqual(s.calls.expired, [false]);
});

test("register adopts the fresh account the same way login does", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["dashboard"], { total: 1 });
  const s = fakeSetters();
  const actions = renderActions(qc, s);
  await withAuthStubs({ register: async () => ({ user: { id: "u3" } }) }, async () => {
    await actions.register({ email: "n@b.c" }, null);
  });
  assert.equal(qc.getQueryData(["dashboard"]), undefined);
  assert.deepEqual(s.calls.user, [{ id: "u3" }]);
});

test("logout clears local state even when the API call fails", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices"], [{ id: "i1" }]);
  const s = fakeSetters();
  const actions = renderActions(qc, s);
  await withAuthStubs({ logout: async () => { throw new Error("network"); } }, async () => {
    await assert.rejects(actions.logout(), /network/);
  });
  assert.deepEqual(s.calls.user, [null]);
  assert.deepEqual(s.calls.expired, [false]);
  assert.equal(qc.getQueryData(["invoices"]), undefined);
});

test("updateProfile refreshes the cached user without touching expiry", async () => {
  const s = fakeSetters();
  const actions = renderActions(new QueryClient(), s);
  await withAuthStubs({ updateProfile: async () => ({ user: { id: "u1", name: "New" } }) }, async () => {
    assert.deepEqual(await actions.updateProfile({ name: "New" }), { id: "u1", name: "New" });
  });
  assert.deepEqual(s.calls.user, [{ id: "u1", name: "New" }]);
  assert.deepEqual(s.calls.expired, []);
});

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

const { orgsKey, useOrgMe, useOrgMembers, useOrgInvites, useActivateOrg, useAcceptInvite } =
  await import("./useOrgs.js");
const { orgsApi } = await import("@/api/orgs");

// Hooks render once via SSR to read their config without a DOM test environment.
function renderHook(client, useHook) {
  let captured;
  function Probe() {
    captured = useHook();
    return null;
  }
  renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(Probe)));
  return captured;
}

test("orgsKey namespaces every org segment", () => {
  assert.deepEqual(orgsKey(), ["orgs"]);
  assert.deepEqual(orgsKey("me"), ["orgs", "me"]);
  assert.deepEqual(orgsKey("members", 2), ["orgs", "members", 2]);
});

test("all five hooks are exported functions", () => {
  for (const fn of [useOrgMe, useOrgMembers, useOrgInvites, useActivateOrg, useAcceptInvite]) {
    assert.equal(typeof fn, "function");
  }
});

test("query hooks register their cache key", () => {
  const qc = new QueryClient();
  renderHook(qc, useOrgMe);
  renderHook(qc, useOrgMembers);
  renderHook(qc, useOrgInvites);
  for (const key of [orgsKey("me"), orgsKey("members"), orgsKey("invites")]) {
    assert.ok(qc.getQueryCache().find({ queryKey: key }), key.join("/"));
  }
});

test("useActivateOrg activates the org and clears the query cache", async () => {
  const qc = new QueryClient();
  qc.setQueryData(orgsKey("me"), { id: "org-1" });
  let clears = 0;
  qc.clear = () => {
    clears += 1;
    QueryClient.prototype.clear.call(qc);
  };
  const original = orgsApi.activate;
  orgsApi.activate = async (id) => ({ id });
  try {
    const mutation = renderHook(qc, useActivateOrg);
    assert.deepEqual(await mutation.mutateAsync("org-2"), { id: "org-2" });
    assert.equal(clears, 1);
    assert.equal(qc.getQueryData(orgsKey("me")), undefined);
  } finally {
    orgsApi.activate = original;
  }
});

test("useAcceptInvite accepts the token and revalidates org queries", async () => {
  const qc = new QueryClient();
  qc.setQueryData(orgsKey("members"), [{ id: "m1" }]);
  const original = orgsApi.accept;
  let payload;
  orgsApi.accept = async (body) => ((payload = body), body);
  try {
    const mutation = renderHook(qc, useAcceptInvite);
    assert.deepEqual(await mutation.mutateAsync("invite-token"), { token: "invite-token" });
    assert.deepEqual(payload, { token: "invite-token" });
    const query = qc.getQueryCache().find({ queryKey: orgsKey("members") });
    assert.equal(query.state.isInvalidated, true);
  } finally {
    orgsApi.accept = original;
  }
});

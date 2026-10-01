import { test } from "node:test";
import assert from "node:assert/strict";
import { QueryClient } from "@tanstack/react-query";

import { installResolver, renderActions, fakeSetters, withAuthStubs } from "./authActionsHarness.js";

installResolver();
const { useAuthActions } = await import("./useAuthActions.js");
const { authApi } = await import("@/api/auth");

test("register adopts the fresh account the same way login does", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["dashboard"], { total: 1 });
  const s = fakeSetters();
  const actions = renderActions(qc, s, useAuthActions);
  await withAuthStubs(authApi, { register: async () => ({ user: { id: "u3" } }) }, async () => {
    await actions.register({ email: "n@b.c" }, null);
  });
  assert.equal(qc.getQueryData(["dashboard"]), undefined);
  assert.deepEqual(s.calls.user, [{ id: "u3" }]);
});

test("register does not adopt a pending account", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["dashboard"], { total: 1 });
  const s = fakeSetters();
  const actions = renderActions(qc, s, useAuthActions);
  let result;
  await withAuthStubs(authApi, { register: async () => ({ status: "verification_required", message: "verification email sent" }) }, async () => {
    result = await actions.register({ email: "p@b.c" }, null);
  });
  assert.deepEqual(result, { verificationRequired: true });
  assert.deepEqual(s.calls.user, [], "a pending account must not be set as the session user");
  assert.equal(qc.getQueryData(["dashboard"]), undefined, "cache is still cleared before the swap");
});

import { test } from "node:test";
import assert from "node:assert/strict";
import { QueryClient } from "@tanstack/react-query";

import { installResolver, renderActions, fakeSetters, withAuthStubs } from "./authActionsHarness.js";

installResolver();
const { useAuthActions } = await import("./useAuthActions.js");
const { authApi } = await import("@/api/auth");

test("useAuthActions returns the five auth actions", () => {
  const actions = renderActions(new QueryClient(), fakeSetters(), useAuthActions);
  for (const name of ["refresh", "login", "register", "logout", "updateProfile"]) {
    assert.equal(typeof actions[name], "function", name);
  }
});

test("refresh stores the session user and stops loading", async () => {
  const s = fakeSetters();
  const actions = renderActions(new QueryClient(), s, useAuthActions);
  await withAuthStubs(authApi, { me: async () => ({ user: { id: "u1" } }) }, async () => {
    await actions.refresh();
  });
  assert.deepEqual(s.calls.user, [{ id: "u1" }]);
  assert.deepEqual(s.calls.expired, [false]);
  assert.deepEqual(s.calls.loading, [false]);
});

test("refresh drops the user when there is no session", async () => {
  const s = fakeSetters();
  const actions = renderActions(new QueryClient(), s, useAuthActions);
  await withAuthStubs(authApi, { me: async () => { throw new Error("401"); } }, async () => {
    await actions.refresh();
  });
  assert.deepEqual(s.calls.user, [null]);
  assert.deepEqual(s.calls.loading, [false]);
});

test("login clears the previous account's cache before swapping identity", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices"], [{ id: "i1" }]);
  const s = fakeSetters();
  const actions = renderActions(qc, s, useAuthActions);
  await withAuthStubs(authApi, { login: async () => ({ user: { id: "u2" } }) }, async () => {
    assert.deepEqual(await actions.login({ email: "a@b.c" }, "cap"), { id: "u2" });
  });
  assert.equal(qc.getQueryData(["invoices"]), undefined);
  assert.deepEqual(s.calls.user, [{ id: "u2" }]);
  assert.deepEqual(s.calls.expired, [false]);
});

test("logout clears local state even when the API call fails", async () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices"], [{ id: "i1" }]);
  const s = fakeSetters();
  const actions = renderActions(qc, s, useAuthActions);
  await withAuthStubs(authApi, { logout: async () => { throw new Error("network"); } }, async () => {
    await assert.rejects(actions.logout(), /network/);
  });
  assert.deepEqual(s.calls.user, [null]);
  assert.deepEqual(s.calls.expired, [false]);
  assert.equal(qc.getQueryData(["invoices"]), undefined);
});

test("updateProfile refreshes the cached user without touching expiry", async () => {
  const s = fakeSetters();
  const actions = renderActions(new QueryClient(), s, useAuthActions);
  await withAuthStubs(authApi, { updateProfile: async () => ({ user: { id: "u1", name: "New" } }) }, async () => {
    assert.deepEqual(await actions.updateProfile({ name: "New" }), { id: "u1", name: "New" });
  });
  assert.deepEqual(s.calls.user, [{ id: "u1", name: "New" }]);
  assert.deepEqual(s.calls.expired, []);
});

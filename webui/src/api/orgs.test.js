import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { readFile } from "node:fs/promises";

// Stub the shared http module so orgs.js runs under plain `node --test`.
const stubUrl =
  "data:text/javascript," +
  encodeURIComponent(`
export const calls = [];
let response = { data: {} };
export function setResponse(next) { response = next; }
export function reset() { calls.length = 0; response = { data: {} }; }
const reply = () => Promise.resolve(response);
export const apiClient = {
  get: (...args) => (calls.push(["get", ...args]), reply()),
  post: (...args) => (calls.push(["post", ...args]), reply()),
  patch: (...args) => (calls.push(["patch", ...args]), reply()),
  delete: (...args) => (calls.push(["delete", ...args]), reply()),
};
`);

register(
  `data:text/javascript,${encodeURIComponent(`
export async function resolve(specifier, context, next) {
  const fromOrgs = (context.parentURL || "").includes("orgs.js");
  if (fromOrgs && (specifier === "./http" || specifier === "@/api/http")) {
    return { url: ${JSON.stringify(stubUrl)}, shortCircuit: true };
  }
  return next(specifier, context);
}
`)}`,
  import.meta.url
);

const stub = await import(stubUrl);
const { orgsApi } = await import("./orgs.js");

test("orgsApi exposes every endpoint method", () => {
  for (const name of ["me", "members", "invites", "invite", "revoke", "accept", "activate", "rename"]) {
    assert.equal(typeof orgsApi[name], "function", name);
  }
});

test("me() and members() hit the org read endpoints", async () => {
  stub.reset();
  stub.setResponse({ data: { org: { id: "o1" } } });
  assert.deepEqual(await orgsApi.me(), { id: "o1" });
  stub.setResponse({ data: { members: [{ id: "m1" }] } });
  assert.deepEqual(await orgsApi.members(), [{ id: "m1" }]);
  assert.deepEqual(stub.calls, [
    ["get", "/orgs/me"],
    ["get", "/orgs/members"],
  ]);
});

test("invites() lists pending invites from /orgs/invites", async () => {
  stub.reset();
  stub.setResponse({ data: { invites: [{ id: "i1", email: "a@b.c" }] } });
  assert.deepEqual(await orgsApi.invites(), [{ id: "i1", email: "a@b.c" }]);
  assert.deepEqual(stub.calls, [["get", "/orgs/invites"]]);
});

test("invite() POSTs the payload to /orgs/invites", async () => {
  stub.reset();
  stub.setResponse({ data: { invite: { id: "i1" } } });
  const payload = { email: "a@b.c", role: "member" };
  assert.deepEqual(await orgsApi.invite(payload), { id: "i1" });
  assert.deepEqual(stub.calls, [["post", "/orgs/invites", payload]]);
});

test("revoke() DELETEs the invite by id", async () => {
  stub.reset();
  assert.equal(await orgsApi.revoke("i9"), undefined);
  assert.deepEqual(stub.calls, [["delete", "/orgs/invites/i9"]]);
});

test("accept() POSTs the acceptance payload to /orgs/invites/accept", async () => {
  stub.reset();
  stub.setResponse({ data: { org: { id: "o2" } } });
  const payload = { token: "t1" };
  assert.deepEqual(await orgsApi.accept(payload), { id: "o2" });
  assert.deepEqual(stub.calls, [["post", "/orgs/invites/accept", payload]]);
});

test("activate() and rename() address the org by id", async () => {
  stub.reset();
  stub.setResponse({ data: { org: { id: "o3", status: "active" } } });
  await orgsApi.activate("o3");
  const payload = { name: "New name" };
  assert.deepEqual(await orgsApi.rename("o3", payload), { id: "o3", status: "active" });
  assert.deepEqual(stub.calls, [
    ["post", "/orgs/o3/activate"],
    ["patch", "/orgs/o3", payload],
  ]);
});

test("orgs.js imports only the shared http client, never axios", async () => {
  const source = await readFile(new URL("./orgs.js", import.meta.url), "utf8");
  assert.doesNotMatch(source, /from\s+["']axios["']/);
  assert.match(source, /import\s*{\s*apiClient\s*}\s*from\s*["'](?:\.\/http|@\/api\/http)["']/);
});

import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";

// Stub the shared http module so auth.js runs under plain `node --test`.
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
  const fromAuth = (context.parentURL || "").includes("authVerify.js");
  if (fromAuth && (specifier === "./http" || specifier === "@/api/http")) {
    return { url: ${JSON.stringify(stubUrl)}, shortCircuit: true };
  }
  return next(specifier, context);
}
`)}`,
  import.meta.url
);

const stub = await import(stubUrl);
const { authVerifyApi } = await import("./authVerify.js");

test("verifyEmail() POSTs the token to /auth/verify-email", async () => {
  stub.reset();
  stub.setResponse({ data: { user: { id: "u1" } } });
  assert.deepEqual(await authVerifyApi.verifyEmail({ token: "t1" }), { user: { id: "u1" } });
  assert.deepEqual(stub.calls, [["post", "/auth/verify-email", { token: "t1" }]]);
});

test("resendVerification() POSTs the email and forwards the captcha header", async () => {
  stub.reset();
  stub.setResponse({ data: { message: "sent" } });
  await authVerifyApi.resendVerification({ email: "a@b.c" }, "cap");
  assert.deepEqual(stub.calls, [
    ["post", "/auth/verify-email/resend", { email: "a@b.c" }, { headers: { "X-Captcha-Token": "cap" } }],
  ]);
});

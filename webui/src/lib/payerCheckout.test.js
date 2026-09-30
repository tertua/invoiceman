import { test } from "node:test";
import assert from "node:assert/strict";
import { openCheckout, usesSnap } from "./payerCheckout.js";

// Minimal window/document doubles: openCheckout must never reach the real
// browser global from a node process, so every access is stubbed explicitly.
function stubWindow() {
  const previous = globalThis.window;
  const previousDocument = globalThis.document;
  const location = { href: "http://pay.test/current" };
  globalThis.window = { location };
  globalThis.document = { head: { appendChild() {} }, createElement: () => ({}) };
  return {
    location,
    restore() {
      globalThis.window = previous;
      globalThis.document = previousDocument;
    },
  };
}

// Records the calls a fake SDK receives; stands in for window.snap.
function fakeSnap() {
  const calls = [];
  return {
    calls,
    pay(token, callbacks) {
      calls.push({ token, callbacks });
    },
  };
}

test("snap checkout loads the injected loader and never touches window.location", async () => {
  const env = stubWindow();
  try {
    const snap = fakeSnap();
    const loaded = [];
    const mode = await openCheckout({
      token: "snap-token-1",
      config: { checkout: "snap", client_key: "ck-1", is_production: true },
      onSuccess: () => {},
      onClose: () => {},
      onError: () => {},
      loader: (isProduction) => {
        loaded.push(isProduction);
        return Promise.resolve(snap);
      },
    });
    assert.equal(mode, "snap");
    assert.deepEqual(loaded, [true], "loader receives the provider-declared environment flag");
    assert.equal(snap.calls.length, 1);
    assert.equal(snap.calls[0].token, "snap-token-1");
    assert.equal(typeof snap.calls[0].callbacks.onSuccess, "function");
    assert.equal(env.location.href, "http://pay.test/current", "no redirect for a Snap checkout");
  } finally {
    env.restore();
  }
});

test("callback wiring survives into the SDK pay call", async () => {
  const env = stubWindow();
  try {
    const snap = fakeSnap();
    let succeeded = 0;
    await openCheckout({
      token: "snap-token-2",
      config: { checkout: "snap" },
      onSuccess: () => { succeeded += 1; },
      loader: () => Promise.resolve(snap),
    });
    snap.calls[0].callbacks.onSuccess();
    assert.equal(succeeded, 1);
  } finally {
    env.restore();
  }
});

test("a redirect_url config assigns window.location.href", async () => {
  const env = stubWindow();
  try {
    const mode = await openCheckout({
      token: "",
      config: { checkout: "hosted", redirect_url: "https://gateway.test/pay/1" },
      onError: () => assert.fail("onError must not fire when a redirect exists"),
    });
    assert.equal(mode, "redirect");
    assert.equal(env.location.href, "https://gateway.test/pay/1");
  } finally {
    env.restore();
  }
});

test("a payment_url config assigns window.location.href", async () => {
  const env = stubWindow();
  try {
    const mode = await openCheckout({
      token: "",
      config: { checkout: "hosted", payment_url: "https://gateway.test/invoice/2" },
      onError: () => assert.fail("onError must not fire when a redirect exists"),
    });
    assert.equal(mode, "redirect");
    assert.equal(env.location.href, "https://gateway.test/invoice/2");
  } finally {
    env.restore();
  }
});

test("a legacy snap config (no checkout key, client_key present) opens Snap", async () => {
  const env = stubWindow();
  try {
    const snap = fakeSnap();
    const mode = await openCheckout({
      token: "snap-legacy",
      config: { client_key: "ck-legacy", is_production: false },
      loader: () => Promise.resolve(snap),
    });
    assert.equal(mode, "snap");
    assert.equal(snap.calls[0].token, "snap-legacy");
    assert.equal(env.location.href, "http://pay.test/current");
  } finally {
    env.restore();
  }
});

test("with neither a snap token nor a hosted url it reports through onError", async () => {
  const env = stubWindow();
  try {
    let reported = null;
    const mode = await openCheckout({
      token: "",
      config: { checkout: "hosted" },
      onError: (err) => { reported = err; },
    });
    assert.equal(mode, "error");
    assert.ok(reported instanceof Error);
    assert.equal(env.location.href, "http://pay.test/current");
  } finally {
    env.restore();
  }
});

test("a token without a snap config redirects instead of loading the SDK", async () => {
  const env = stubWindow();
  try {
    const mode = await openCheckout({
      token: "hosted-token",
      config: { checkout: "hosted", redirect_url: "https://gateway.test/pay/3" },
      loader: () => assert.fail("the snap loader must not be called for a hosted checkout"),
      onError: () => assert.fail("onError must not fire when a redirect exists"),
    });
    assert.equal(mode, "redirect");
    assert.equal(env.location.href, "https://gateway.test/pay/3");
  } finally {
    env.restore();
  }
});

test("usesSnap only trusts the provider declaration or a legacy client key", () => {
  assert.equal(usesSnap({ token: "t", config: { checkout: "snap" } }), true);
  assert.equal(usesSnap({ token: "t", config: { checkout: "hosted" } }), false);
  assert.equal(usesSnap({ token: "t", config: { client_key: "ck" } }), true);
  assert.equal(usesSnap({ token: "t", config: {} }), false);
  assert.equal(usesSnap({ token: "", config: { checkout: "snap" } }), false);
  assert.equal(usesSnap({ token: "t" }), false);
  assert.equal(usesSnap(), false);
});

import { test } from "node:test";
import assert from "node:assert/strict";
import { finishState } from "./paymentFinish.js";

test("midtrans success statuses resolve to success", () => {
  assert.equal(finishState("settlement"), "success");
  assert.equal(finishState("capture"), "success");
});

test("midtrans in-flight statuses resolve to pending", () => {
  assert.equal(finishState("pending"), "pending");
  assert.equal(finishState("challenge"), "pending");
});

test("midtrans terminal failure statuses resolve to failed", () => {
  assert.equal(finishState("deny"), "failed");
  assert.equal(finishState("cancel"), "failed");
  assert.equal(finishState("expire"), "failed");
  assert.equal(finishState("failure"), "failed");
});

test("matching is case-insensitive and trims whitespace", () => {
  assert.equal(finishState("Settlement"), "success");
  assert.equal(finishState("  PENDING "), "pending");
});

test("missing or foreign statuses resolve to unknown", () => {
  assert.equal(finishState(null), "unknown");
  assert.equal(finishState(undefined), "unknown");
  assert.equal(finishState(""), "unknown");
  assert.equal(finishState("bogus"), "unknown");
});

import { test } from "node:test";
import assert from "node:assert/strict";
import { resolvePayStep } from "./payStep.js";

test("a paid invoice is always the done step", () => {
  assert.equal(resolvePayStep({ isPaid: true, canPay: true, panelStep: "choose" }), "done");
  assert.equal(resolvePayStep({ isPaid: true, canPay: false, panelStep: "pay" }), "done");
});

test("an unpaid invoice follows the panel step", () => {
  assert.equal(resolvePayStep({ isPaid: false, canPay: true, panelStep: "choose" }), "choose");
  assert.equal(resolvePayStep({ isPaid: false, canPay: true, panelStep: "pay" }), "pay");
  assert.equal(resolvePayStep({ isPaid: false, canPay: true, panelStep: "done" }), "done");
});

test("a non-payable invoice stays on choose regardless of stale panel state", () => {
  assert.equal(resolvePayStep({ isPaid: false, canPay: false, panelStep: "pay" }), "choose");
});

test("missing panel step defaults to choose", () => {
  assert.equal(resolvePayStep({ isPaid: false, canPay: true }), "choose");
  assert.equal(resolvePayStep({}), "choose");
});

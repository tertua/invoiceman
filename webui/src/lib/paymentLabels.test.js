import { test } from "node:test";
import assert from "node:assert/strict";
import { paymentMethodLabel } from "./paymentLabels.js";

test("legacy provider display names map to public-pay labels", () => {
  assert.equal(paymentMethodLabel("Midtrans"), "QRIS");
  assert.equal(paymentMethodLabel("NOWPayments"), "Crypto");
});

test("matching is case-insensitive and trims whitespace", () => {
  assert.equal(paymentMethodLabel("  midtrans "), "QRIS");
  assert.equal(paymentMethodLabel("nowpayments"), "Crypto");
});

test("manual methods pass through unchanged", () => {
  assert.equal(paymentMethodLabel("Cash"), "Cash");
  assert.equal(paymentMethodLabel("Bank transfer"), "Bank transfer");
  assert.equal(paymentMethodLabel("Online"), "Online");
});

test("missing or non-string methods resolve to empty", () => {
  assert.equal(paymentMethodLabel(null), "");
  assert.equal(paymentMethodLabel(undefined), "");
  assert.equal(paymentMethodLabel(""), "");
  assert.equal(paymentMethodLabel("   "), "");
});

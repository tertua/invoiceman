import { apiClient } from "./http";

// Idempotent public intent creation (money path). The caller passes one
// UUID per user intent as Idempotency-Key; a double-fired request (StrictMode
// remount, double-tap, retry) then replays the first charge instead of
// opening a second one at the gateway. Kept out of publicPay.js so the
// baseline-locked seam file never grows.
export function createPublicTransaction(token, method, extra, idempotencyKey) {
  const body = { ...(method ? { payment_method: method } : {}), ...(extra || {}) };
  const config = idempotencyKey ? { headers: { "Idempotency-Key": idempotencyKey } } : undefined;
  return apiClient.post(`/public/pay/${token}/transaction`, body, config).then((r) => r.data);
}

// One UUID per QR generation, with a non-crypto fallback for old webviews.
export function newIntentKey(nonce) {
  try {
    return crypto.randomUUID();
  } catch {
    return `${Date.now()}-${Math.random().toString(36).slice(2)}-${nonce}`;
  }
}

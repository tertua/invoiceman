// Legacy provider display names -> neutral method labels. This is the webui
// mirror of the backend fallback `gateway.ProviderLegacyLabels()`
// (platform/gateway/payment_methods.go, served by
// app/controllers/public_method_label.go): rows settled through a gateway keep
// the provider display name in payments.method, so the owner-facing views map
// it back to the same neutral label the public pay page shows. It labels
// historical data ONLY — live labels come from the transaction's method id —
// so it is intentionally the one place a provider name is spelled out.
export const LEGACY_PROVIDER_LABELS = {
  midtrans: "QRIS",
  nowpayments: "Crypto",
};

// paymentMethodLabel maps a stored payments.method value to its display label.
// A legacy provider name resolves through LEGACY_PROVIDER_LABELS; anything else
// (manual methods, a neutral method id) passes through unchanged. Asset
// suffixes ("Crypto · USDT (BEP20)") need the transaction row and stay BE-only.
export function paymentMethodLabel(method) {
  const raw = typeof method === "string" ? method.trim() : "";
  if (!raw) return "";
  return LEGACY_PROVIDER_LABELS[raw.toLowerCase()] || raw;
}

// Mirrors the backend fallback in app/controllers/public_method_label.go
// (payMethodLabel without a transaction): rows settled through a gateway keep
// the provider display name in payments.method, so the owner-facing views map
// it back to the same neutral label the public pay page shows. Asset suffixes
// ("Crypto · USDT (BEP20)") need the transaction row and stay BE-only.
export function paymentMethodLabel(method) {
  const raw = typeof method === "string" ? method.trim() : "";
  if (!raw) return "";
  const key = raw.toLowerCase();
  if (key === "midtrans") return "QRIS";
  if (key === "nowpayments") return "Crypto";
  return raw;
}

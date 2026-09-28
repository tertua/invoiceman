// Maps a payment-provider redirect status (Midtrans `transaction_status`
// appended to the Finish Redirect URL) to a static thank-you page state.
// Pure and dependency-free so it stays unit-testable with node:test.
export function finishState(status) {
  const s = String(status ?? "").toLowerCase().trim();
  if (s === "settlement" || s === "capture") return "success";
  if (s === "pending" || s === "challenge") return "pending";
  if (s === "deny" || s === "cancel" || s === "expire" || s === "failure") return "failed";
  return "unknown";
}

// Crypto asset list for the crypto widget. Codes must match the backend
// allowlist in platform/nowpayments/assets.go; labels are display-only.
export const CRYPTO_ASSETS = [
  { code: "usdttrc20", ticker: "USDT", network: "TRC20" },
  { code: "usdterc20", ticker: "USDT", network: "ERC20" },
  { code: "usdtbsc", ticker: "USDT", network: "BEP20" },
  { code: "trx", ticker: "TRX", network: "" },
  { code: "doge", ticker: "DOGE", network: "" },
  { code: "ltc", ticker: "LTC", network: "" },
];

export const DEFAULT_CRYPTO_ASSET = "usdtbsc";

export function cryptoLabel(payCurrency) {
  const c = String(payCurrency || "").toLowerCase();
  const found = CRYPTO_ASSETS.find((a) => a.code === c);
  if (found) return found.network ? `${found.ticker} (${found.network})` : found.ticker;
  return String(payCurrency || "USDT").toUpperCase();
}

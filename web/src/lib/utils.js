import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { t, LOCALES } from "./i18n";

export function cn(...inputs) {
  return twMerge(clsx(inputs));
}

let currentLang = "en";
let currentLocale = "en-US";
let currentCurrency = "IDR";

export function setLocale(lang) {
  currentLang = lang === "id" ? "id" : "en";
  currentLocale = LOCALES[currentLang];
}

export function setDefaultCurrency(currency) {
  if (currency) currentCurrency = currency;
}

function parseDate(date) {
  if (typeof date === "string" && /^\d{4}-\d{2}-\d{2}$/.test(date)) {
    return new Date(`${date}T00:00:00`);
  }
  return typeof date === "string" ? new Date(date) : date;
}

export function formatNumber(n, opts = {}) {
  return new Intl.NumberFormat(currentLocale, opts).format(n);
}

export const CURRENCIES = [
  { code: "IDR", symbol: "Rp" },
  { code: "USD", symbol: "$" },
  { code: "EUR", symbol: "€" },
  { code: "GBP", symbol: "£" },
  { code: "INR", symbol: "₹" },
  { code: "CAD", symbol: "$" },
  { code: "AUD", symbol: "$" },
  { code: "JPY", symbol: "¥" },
];

export function formatMoney(amount, currency = currentCurrency) {
  const n = Number(amount) || 0;
  const cur = currency || currentCurrency;
  const noFraction = cur === "IDR";
  const locale = cur === "IDR" ? "id-ID" : currentLocale;
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency: cur,
      minimumFractionDigits: noFraction ? 0 : 2,
      maximumFractionDigits: noFraction ? 0 : 2,
    }).format(n);
  } catch {
    return `$${n.toFixed(2)}`;
  }
}

export function formatDate(date, opts = { month: "short", day: "numeric", year: "numeric" }) {
  if (!date) return "—";
  const d = parseDate(date);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString(currentLocale, opts);
}

export function formatMonthShort(date) {
  if (!date) return "";
  const d = parseDate(date);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleDateString(currentLocale, { month: "short" });
}

// Convert any date to a YYYY-MM-DD string for <input type="date"> / API.
export function toDateInput(date) {
  if (!date) return "";
  if (typeof date === "string" && /^\d{4}-\d{2}-\d{2}$/.test(date)) return date;
  const d = parseDate(date);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function todayDateInput() {
  return toDateInput(new Date());
}

export function addDaysDateInput(days) {
  const date = new Date();
  date.setDate(date.getDate() + days);
  return toDateInput(date);
}

export function relativeTime(date) {
  if (!date) return "";
  const d = parseDate(date);
  if (!(d instanceof Date) || Number.isNaN(d.getTime())) return "";
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return t(currentLang, "relative.justNow");
  if (diff < 3600) return t(currentLang, "relative.minAgo", { n: Math.floor(diff / 60) });
  if (diff < 86400) return t(currentLang, "relative.hourAgo", { n: Math.floor(diff / 3600) });
  if (diff < 604800) return t(currentLang, "relative.dayAgo", { n: Math.floor(diff / 86400) });
  return d.toLocaleDateString(currentLocale);
}

// Stable AR-aging bucket keys returned by GET /api/reports.
export const AGING_BUCKET_KEYS = ["current", "d1_30", "d31_60", "d61_90", "d90_plus"];

// Legacy display labels once sent by the API; mapped so cached responses
// still resolve to a translatable key.
const LEGACY_AGING_LABELS = {
  Current: "current",
  "1-30 days": "d1_30",
  "31-60 days": "d31_60",
  "61-90 days": "d61_90",
  "90+ days": "d90_plus",
};

export function agingBucketKey(point) {
  const raw = point?.key || point?.bucket;
  if (!raw) return "current";
  if (AGING_BUCKET_KEYS.includes(raw)) return raw;
  return LEGACY_AGING_LABELS[raw] || raw;
}

// Replace each point's chart label with its localized text. Must never
// throw on missing/invalid input — fall back to the raw value.
export function localizeAgingBuckets(aging, translate) {
  if (!Array.isArray(aging)) return [];
  return aging.map((a) => {
    if (!a || typeof a !== "object") return a;
    const key = agingBucketKey(a);
    let label = a.bucket;
    try {
      label = translate(`aging.${key}`);
    } catch {
      /* keep raw label */
    }
    return { ...a, key, bucket: label || a.bucket };
  });
}

// Monthly points carry a stable "YYYY-MM" key; render the short month name
// in the active locale. Must never throw — fall back to the raw label.
export function monthKeyToLabel(key, fallback) {
  if (typeof key !== "string" || !/^\d{4}-\d{2}$/.test(key)) return fallback ?? "";
  const year = Number(key.slice(0, 4));
  const month = Number(key.slice(5, 7));
  if (month < 1 || month > 12) return fallback ?? "";
  const d = new Date(year, month - 1, 1);
  if (Number.isNaN(d.getTime())) return fallback ?? "";
  try {
    return d.toLocaleDateString(currentLocale, { month: "short" });
  } catch {
    return fallback ?? "";
  }
}

export function localizeMonthLabels(rows) {
  if (!Array.isArray(rows)) return [];
  return rows.map((r) => {
    if (!r || typeof r !== "object") return r;
    if (!r.key) return r;
    return { ...r, label: monthKeyToLabel(r.key, r.label) };
  });
}

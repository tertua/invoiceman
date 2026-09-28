import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { t, LOCALES } from "./i18n";

export { AGING_BUCKET_KEYS, agingBucketKey, localizeAgingBuckets } from "./chartLabels";

export function cn(...inputs) {
  return twMerge(clsx(inputs));
}

function resolveInitialLang() {
  try {
    const stored = localStorage.getItem("arr-lang");
    if (stored === "en" || stored === "id") return stored;
  } catch {
    /* storage unavailable */
  }
  try {
    if (navigator.language?.toLowerCase().startsWith("id")) return "id";
  } catch {
    /* navigator unavailable */
  }
  return "en";
}

let currentLang = resolveInitialLang();
let currentLocale = LOCALES[currentLang];
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
  const n = Number.parseFloat(amount) || 0;
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

// Backend error messages are English-only; each known message has an
// `api.<message>` dictionary entry in i18n.js. Unknown/dynamic messages
// (e.g. validator details, JWT library errors) pass through untouched.
// Must never throw — fall back to the raw message.
export function localizeApiError(message) {
  if (typeof message !== "string" || !message) return message;
  try {
    const key = `api.${message}`;
    const localized = t(currentLang, key);
    return localized === key ? message : localized;
  } catch {
    return message;
  }
}

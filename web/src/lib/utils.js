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

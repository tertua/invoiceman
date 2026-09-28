// i18n — EN/ID dictionary entrypoint.
// Dictionaries live in `i18n.en.js` / `i18n.id.js` so editors and AI
// assistants load one language (~800 lines) instead of the full file.
// `en` is the source of truth; any missing `id` key falls back to `en`.
// Usage: t(lang, "key", { vars }) — `{name}` tokens are interpolated.

import { en } from "./i18n.en.js";
import { id } from "./i18n.id.js";

export const LOCALES = { en: "en-US", id: "id-ID" };

export const translations = { en, id };

export function t(lang, key, vars) {
  const dict = translations[lang] || translations.en;
  let s = dict[key] ?? translations.en[key] ?? key;
  if (vars) {
    for (const [k, v] of Object.entries(vars)) {
      s = s.replaceAll(`{${k}}`, String(v ?? ""));
    }
  }
  return s;
}

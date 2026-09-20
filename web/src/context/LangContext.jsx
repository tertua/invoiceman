import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { t as translate } from "@/lib/i18n";
import { setLocale } from "@/lib/utils";

const LangContext = createContext(null);

const STORAGE_KEY = "arr-lang";

function resolveInitial() {
  if (typeof window === "undefined") return "en";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "en" || stored === "id") return stored;
  return "en";
}

export function LangProvider({ children }) {
  const [lang, setLang] = useState(resolveInitial);

  useEffect(() => {
    document.documentElement.setAttribute("lang", lang);
    localStorage.setItem(STORAGE_KEY, lang);
    setLocale(lang);
  }, [lang]);

  const t = useCallback((key, vars) => translate(lang, key, vars), [lang]);

  const value = useMemo(() => ({ lang, setLang, t }), [lang, t]);

  return <LangContext.Provider value={value}>{children}</LangContext.Provider>;
}

export function useLang() {
  const ctx = useContext(LangContext);
  if (!ctx) throw new Error("useLang must be used inside LangProvider");
  return ctx;
}

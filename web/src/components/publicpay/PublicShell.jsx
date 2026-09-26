import { Link } from "react-router-dom";
import { useAppName } from "@/hooks/useConfig";

export function LangToggle({ lang, onLang }) {
  return (
    <div className="flex items-center gap-1 rounded-full border border-[var(--border)] p-1 text-[11px] font-semibold">
      {["en", "id"].map((code) => (
        <button
          key={code}
          type="button"
          onClick={() => onLang(code)}
          aria-pressed={lang === code}
          className={`px-2.5 py-1 rounded-full uppercase tracking-wide transition-colors ${
            lang === code ? "bg-[var(--ink)] text-[var(--bg)]" : "text-[var(--ink-muted)] hover:text-[var(--ink)]"
          }`}
        >
          {code}
        </button>
      ))}
    </div>
  );
}

export default function PublicShell({ children, branding, lang, onLang }) {
  const appName = useAppName();
  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-6 p-6 bg-[var(--bg)]">
      <div className="w-full max-w-[520px] flex items-center justify-between gap-3">
        <div className="flex items-center gap-2.5 min-w-0 flex-1">
          {branding?.logo_url ? (
            <img src={branding.logo_url} alt={branding.company_name || "logo"} className="h-9 w-9 object-contain rounded shrink-0" />
          ) : null}
          {branding?.company_name ? (
            <div className="font-display text-base font-semibold text-[var(--ink)] truncate">{branding.company_name}</div>
          ) : null}
        </div>
        <div className="shrink-0">
          <LangToggle lang={lang} onLang={onLang} />
        </div>
      </div>
      {children}
      <p className="text-[11px] text-[var(--ink-muted)]">
        Powered by{" "}
        <Link to="/" className="font-semibold text-[var(--ink)] hover:underline">
          {appName}
        </Link>
      </p>
    </div>
  );
}

import { ShieldCheck } from "lucide-react";

import { useLang } from "@/context/LangContext";
import { useOidcEnabled } from "@/hooks/useConfig";

// API base mirrors api/http.js baseURL; the SSO button is a plain browser
// navigation (never axios), so the server can drive the whole OIDC redirect.
const API_BASE = "/api/v1";

// OidcButton renders the "or / Sign in with SSO" divider + button when the
// backend advertises OIDC as active, and nothing otherwise. It is a plain
// anchor (full browser navigation, never axios) so the server owns the flow.
export function OidcButton() {
  const { t } = useLang();
  const oidcEnabled = useOidcEnabled();

  if (!oidcEnabled) return null;

  return (
    <div className="mt-8">
      <div className="flex items-center gap-3 text-xs uppercase tracking-wide text-[var(--ink-muted)]">
        <span className="h-px flex-1 bg-[var(--border)]" />
        {t("auth.orContinueWith")}
        <span className="h-px flex-1 bg-[var(--border)]" />
      </div>
      <a
        href={`${API_BASE}/auth/oidc/login`}
        className="mt-4 w-full inline-flex items-center justify-center gap-2 rounded-2xl border border-[var(--border)] px-4 py-2.5 text-sm font-semibold text-[var(--ink)] hover:bg-[var(--accent-soft)]"
      >
        <ShieldCheck size={15} />
        {t("auth.signInWithSso")}
      </a>
    </div>
  );
}

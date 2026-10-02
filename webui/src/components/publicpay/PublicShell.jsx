import { Link } from "react-router-dom";
import { useAppName } from "@/hooks/useConfig";

export default function PublicShell({ children, branding }) {
  const appName = useAppName();
  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-6 p-6 bg-[var(--bg)]">
      <div className="w-full max-w-[520px] flex items-center justify-center gap-2.5">
        {branding?.logo_url ? (
          <img src={branding.logo_url} alt={branding.company_name || "logo"} className="h-9 w-9 object-contain rounded shrink-0" />
        ) : null}
        {branding?.company_name ? (
          <div className="font-display text-base font-semibold text-[var(--ink)] truncate">{branding.company_name}</div>
        ) : null}
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

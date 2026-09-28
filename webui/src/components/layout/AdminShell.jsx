import { useEffect } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { ArrowLeft, ShieldCheck } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLang } from "@/context/LangContext";
import { useAppName } from "@/hooks/useConfig";

function AdminTab({ to, children }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        cn(
          "px-3 py-1.5 rounded-lg text-sm font-medium whitespace-nowrap transition-colors",
          isActive
            ? "bg-[var(--ink)] text-[var(--bg)]"
            : "text-[var(--ink-muted)] hover:text-[var(--ink)] hover:bg-[var(--surface-2)]",
        )
      }
    >
      {children}
    </NavLink>
  );
}

export function AdminShell() {
  const { t } = useLang();
  const appName = useAppName();
  useEffect(() => {
    document.title = `${appName} — ${t("admin.console")}`;
  }, [appName, t]);
  return (
    <div className="min-h-screen flex flex-col bg-[var(--bg)]">
      <header className="border-b border-[var(--border)] bg-[var(--surface)]">
        <div className="mx-auto max-w-[1600px] px-6 md:px-8 h-14 flex items-center justify-between gap-3">
          <div className="flex items-center gap-3 min-w-0 flex-1">
            <span className="flex items-center gap-2 text-sm font-semibold text-[var(--ink)]">
              <ShieldCheck size={16} className="text-[var(--accent)]" />
              {t("admin.console")}
            </span>
          </div>
            <nav className="flex items-center gap-1">
              <AdminTab to="/gateway">{t("admin.gateway")}</AdminTab>
              <AdminTab to="/settlement">{t("gateway.settlement")}</AdminTab>
              <AdminTab to="/users">{t("admin.users")}</AdminTab>
            </nav>
          <div className="flex-1 flex justify-end">
            <a
              href="/dashboard"
              className="flex items-center gap-1.5 text-sm font-medium text-[var(--ink-muted)] hover:text-[var(--ink)] transition-colors shrink-0"
            >
              <ArrowLeft size={16} />
              {t("admin.backToApp")}
            </a>
          </div>
        </div>
      </header>
      <main className="flex-1 px-6 md:px-8 py-6 max-w-[1600px] mx-auto w-full">
        <Outlet />
      </main>
    </div>
  );
}

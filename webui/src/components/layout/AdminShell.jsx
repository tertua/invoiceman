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
          "relative px-4 py-2 rounded-xl text-sm font-medium whitespace-nowrap transition-all duration-200",
          isActive
            ? "bg-[var(--accent)] text-white shadow-sm"
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
      <header className="sticky top-0 z-30 border-b border-[var(--border)] bg-[var(--surface)]/80 backdrop-blur-xl">
        <div className="mx-auto max-w-[1600px] px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <div className="flex items-center gap-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--accent)]/10">
                  <ShieldCheck size={18} className="text-[var(--accent)]" />
                </div>
                <span className="text-base font-semibold text-[var(--ink)]">
                  {t("admin.console")}
                </span>
              </div>
            </div>
            <nav className="hidden md:flex items-center gap-1.5 px-2 py-1 bg-[var(--surface-2)]/50 rounded-2xl">
              <AdminTab to="/gateway">{t("admin.gateway")}</AdminTab>
              <AdminTab to="/settlement">{t("gateway.settlement")}</AdminTab>
              <AdminTab to="/users">{t("admin.users")}</AdminTab>
            </nav>
            <div className="flex items-center">
              <a
                href="/dashboard"
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium text-[var(--ink-muted)] hover:text-[var(--ink)] hover:bg-[var(--surface-2)] transition-all duration-200"
              >
                <ArrowLeft size={16} />
                <span className="hidden sm:inline">{t("admin.backToApp")}</span>
              </a>
            </div>
          </div>
          <nav className="md:hidden flex items-center gap-1.5 pb-3 overflow-x-auto scrollbar-hide">
            <AdminTab to="/gateway">{t("admin.gateway")}</AdminTab>
            <AdminTab to="/settlement">{t("gateway.settlement")}</AdminTab>
            <AdminTab to="/users">{t("admin.users")}</AdminTab>
          </nav>
        </div>
      </header>
      <main className="flex-1 px-4 sm:px-6 lg:px-8 py-8 max-w-[1600px] mx-auto w-full">
        <Outlet />
      </main>
    </div>
  );
}

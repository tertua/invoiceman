import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import { ArrowLeft, ShieldCheck } from "lucide-react";
import { useLang } from "@/context/LangContext";
import { useAppName } from "@/hooks/useConfig";

export function AdminShell() {
  const { t } = useLang();
  const appName = useAppName();
  useEffect(() => {
    document.title = `${appName} — ${t("admin.console")}`;
  }, [appName, t]);
  return (
    <div className="min-h-screen flex flex-col bg-[var(--bg)]">
      <header className="border-b border-[var(--border)] bg-[var(--surface)]">
        <div className="mx-auto max-w-[1600px] px-6 md:px-8 h-14 flex items-center justify-between">
          <span className="flex items-center gap-2 text-sm font-semibold text-[var(--ink)]">
            <ShieldCheck size={16} className="text-[var(--accent)]" />
            {t("admin.console")}
          </span>
          <a
            href="/dashboard"
            className="flex items-center gap-1.5 text-sm font-medium text-[var(--ink-muted)] hover:text-[var(--ink)] transition-colors"
          >
            <ArrowLeft size={16} />
            {t("admin.backToApp")}
          </a>
        </div>
      </header>
      <main className="flex-1 px-6 md:px-8 py-6 max-w-[1600px] mx-auto w-full">
        <Outlet />
      </main>
    </div>
  );
}

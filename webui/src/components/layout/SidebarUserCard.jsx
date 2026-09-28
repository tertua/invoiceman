import { LogOut } from "lucide-react";
import { cn } from "@/lib/utils";

export function UserCard({ user, logout, t }) {
  const displayName = user?.name || t("sidebar.account");
  const displayEmail = user?.email || "";
  return (
    <div
      className={cn(
        "flex items-center h-12 mt-1 w-10 group-hover/sidebar:w-[200px] overflow-hidden",
        "transition-[width] duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]",
      )}
    >
      <div
        className="h-10 w-10 rounded-full bg-[var(--accent-soft)] text-[var(--accent-strong)] font-semibold flex items-center justify-center text-sm ring-2 ring-[var(--surface)] shrink-0">
        {user?.name?.[0]?.toUpperCase() || "R"}
      </div>
      <div
        className={cn(
          "ml-3 min-w-0 flex-1",
          "opacity-0 -translate-x-1",
          "transition-[opacity,transform] duration-200 ease-out",
          "group-hover/sidebar:opacity-100 group-hover/sidebar:translate-x-0 group-hover/sidebar:delay-100",
        )}
      >
        <div className="text-sm font-semibold text-[var(--ink)] truncate">
          {displayName}
        </div>
        {displayEmail && (
          <div className="text-[11px] text-[var(--ink-muted)] truncate">
            {displayEmail}
          </div>
        )}
      </div>
      <button type="button" onClick={logout} title={t("sidebar.logOut")} aria-label={t("sidebar.logOut")}
        className="shrink-0 h-8 w-8 rounded-full hidden group-hover/sidebar:flex items-center justify-center text-[var(--ink-muted)] hover:text-[var(--danger)] hover:bg-[var(--danger)]/10 transition-colors">
        <LogOut size={15} />
      </button>
    </div>
  );
}

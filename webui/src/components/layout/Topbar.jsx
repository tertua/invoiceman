import { Search, Sun, Moon } from "lucide-react";
import { IconButton } from "@/components/ui/IconButton";
import { useTheme } from "@/context/ThemeContext";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";
import { NotificationsPopover } from "./NotificationsPopover";

export function Topbar({ onOpenPalette }) {
  const { theme, toggle } = useTheme();
  const { user } = useAuth();
  const { t } = useLang();
  const firstName = user?.name?.split(" ")[0] || "there";

  const isMac =
    typeof navigator !== "undefined" && /Mac|iPhone|iPad/i.test(navigator.platform);

  return (
    <header className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 sm:gap-6 mb-6">
      <div>
        <h1 className="font-display text-[clamp(28px,3vw,38px)] font-semibold leading-tight text-[var(--ink)]">
          {t("topbar.hello", { name: firstName })}
        </h1>
        <p className="text-sm text-[var(--ink-muted)] mt-1">
          {t("topbar.subtitle")}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2 sm:gap-3 shrink-0">
        <button
          type="button"
          onClick={onOpenPalette}
          className="hidden lg:flex items-center gap-3 h-11 w-[360px] rounded-full bg-[var(--surface)] border border-[var(--border)] pl-5 pr-1.5 shadow-card transition-shadow hover:shadow-hover text-left"
        >
          <Search size={16} className="text-[var(--ink-muted)] shrink-0" />
          <span className="flex-1 text-sm text-[var(--ink-muted)] truncate">
            {t("topbar.searchPlaceholder")}
          </span>
          <kbd className="inline-flex items-center gap-0.5 text-[10px] px-2 h-7 rounded-full bg-[var(--surface-2)] text-[var(--ink-muted)] border border-[var(--border)] font-semibold">
            {isMac ? "⌘" : "Ctrl"} K
          </kbd>
        </button>

        <IconButton
          onClick={onOpenPalette}
          title={t("topbar.search")}
          className="lg:hidden"
        >
          <Search size={16} />
        </IconButton>

        <IconButton onClick={toggle} title={t("topbar.toggleTheme")}>
          {theme === "light" ? <Moon size={16} /> : <Sun size={16} />}
        </IconButton>
        <NotificationsPopover />
      </div>
    </header>
  );
}

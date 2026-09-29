import { useMemo } from "react";
import { useLang } from "@/context/LangContext";
import { cn } from "@/lib/utils";

// Status filter pills for the invoice list; the page owns the selected key.
export function InvoiceStatusTabs({ status, onSelect }) {
  const { t } = useLang();
  const tabs = useMemo(
    () => [
      { key: "all", label: t("invoices.all") },
      { key: "draft", label: t("status.draft") },
      { key: "sent", label: t("status.sent") },
      { key: "paid", label: t("status.paid") },
      { key: "overdue", label: t("status.overdue") },
    ],
    [t]
  );

  return (
    <div className="flex items-center gap-1 p-1 rounded-full bg-[var(--surface)] border border-[var(--border)] shadow-card w-fit">
      {tabs.map((tab) => (
        <button type="button"
          key={tab.key}
          onClick={() => onSelect(tab.key)}
          className={cn(
            "h-8 px-4 rounded-full text-xs font-semibold transition-colors",
            status === tab.key
              ? "bg-[var(--ink)] text-[var(--bg)]"
              : "text-[var(--ink-muted)] hover:text-[var(--ink)]"
          )}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}

import { cn } from "@/lib/utils";

// Invoice status tabs with optional count badges. Extracted from Invoices.jsx
// so the page stays a composition and the tab row can evolve (counts, overflow)
// without touching the list. `counts` is null while loading / on error: the
// badge is simply omitted, never rendered as 0.
export function InvoiceStatusTabs({ tabs, status, onChange, counts, countsError, errorLabel }) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-1 p-1 rounded-full bg-[var(--surface)] border border-[var(--border)] shadow-card w-fit">
        {tabs.map((tab) => (
          <button
            type="button"
            key={tab.key}
            onClick={() => onChange(tab.key)}
            aria-current={status === tab.key ? "page" : undefined}
            className={cn(
              "h-8 px-4 rounded-full text-xs font-semibold transition-colors",
              status === tab.key
                ? "bg-[var(--ink)] text-[var(--bg)]"
                : "text-[var(--ink-muted)] hover:text-[var(--ink)]"
            )}
          >
            {tab.label}
            {counts?.[tab.key] != null && (
              <span className={cn("ml-1.5 tabular", status === tab.key ? "opacity-80" : "text-[var(--ink-muted)]")}>
                {counts[tab.key]}
              </span>
            )}
          </button>
        ))}
      </div>
      {countsError && <span className="text-[11px] text-[var(--ink-muted)] pl-2">{errorLabel}</span>}
    </div>
  );
}

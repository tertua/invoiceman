import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/Card";

// Compact stat tiles that sit above each console page's main table. Mirrors the
// StatCard/StatValue visual language (uppercase label, tabular display-font
// number) but tuned for density — no sparkline, no recharts coupling.
const TONE_CHIP = {
  neutral: "bg-[var(--surface-2)] text-[var(--ink-muted)]",
  accent: "bg-[var(--accent-soft)] text-[var(--accent-strong)]",
  success: "bg-[var(--success)]/12 text-[var(--success)]",
  warning: "bg-[var(--warning)]/12 text-[var(--warning)]",
  danger: "bg-[var(--danger)]/12 text-[var(--danger)]",
};

export function PageStatStrip({ children }) {
  return <div className="mb-5 flex flex-wrap gap-3">{children}</div>;
}

export function PageStat({ label, value, sub, icon: Icon, tone = "neutral" }) {
  return (
    <Card radius="md" padding="sm" className="min-w-[170px] flex-1">
      <div className="flex items-start justify-between gap-2">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-[var(--ink-muted)]">
          {label}
        </span>
        {Icon && (
          <span className={cn("flex h-6 w-6 items-center justify-center rounded-full", TONE_CHIP[tone])}>
            <Icon size={13} />
          </span>
        )}
      </div>
      <div className="mt-1.5 font-display tabular text-xl font-semibold tracking-tight text-[var(--ink)]">
        {value}
      </div>
      {sub && <div className="mt-0.5 text-xs text-[var(--ink-muted)]">{sub}</div>}
    </Card>
  );
}

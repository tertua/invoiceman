import { cn } from "@/lib/utils";

export function StatusPill({ active, t }) {
  return (
    <span className={cn(
      "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold",
      active ? "bg-[var(--success)]/12 text-[var(--success)]" : "bg-[var(--surface-2)] text-[var(--ink-muted)]",
    )}>
      {active ? t("gateway.active") : t("gateway.inactive")}
    </span>
  );
}

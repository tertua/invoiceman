import { cn } from "@/lib/utils";

// One pill language for every console status. `active` keeps the legacy
// projects boolean call; `status` covers the settlement/user status strings.
export const STATUS_TONE = {
  success: "bg-[var(--success)]/12 text-[var(--success)]",
  delivered: "bg-[var(--success)]/12 text-[var(--success)]",
  pending: "bg-[var(--warning)]/12 text-[var(--warning)]",
  failed: "bg-[var(--danger)]/12 text-[var(--danger)]",
  expired: "bg-[var(--surface-2)] text-[var(--ink-muted)]",
  refunded: "bg-[var(--info)]/12 text-[var(--info)]",
  partially_refunded: "bg-[var(--info)]/12 text-[var(--info)]",
};

export function StatusPill({ active, status, t, className }) {
  if (status) {
    const label = t(`status.${status}`);
    return (
      <span
        className={cn(
          "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold capitalize",
          STATUS_TONE[status] || "bg-[var(--surface-2)] text-[var(--ink-muted)]",
          className,
        )}
        title={label}
      >
        {label}
      </span>
    );
  }
  return (
    <span
      className={cn(
        "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold",
        active
          ? "bg-[var(--success)]/12 text-[var(--success)]"
          : "bg-[var(--surface-2)] text-[var(--ink-muted)]",
        className,
      )}
    >
      {active ? t("gateway.active") : t("gateway.inactive")}
    </span>
  );
}

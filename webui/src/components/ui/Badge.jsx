import { cva } from "class-variance-authority";
import { cn } from "@/lib/utils";
import { useLang } from "@/context/LangContext";

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold tracking-tight tabular",
  {
    variants: {
      tone: {
        neutral:
          "bg-[var(--surface-2)] text-[var(--ink-muted)] border border-[var(--border)]",
        accent: "bg-[var(--accent-soft)] text-[var(--accent-strong)]",
        info: "bg-[var(--info)]/16 text-[var(--info)] border border-[var(--info)]/25",
        success: "bg-[var(--success)]/16 text-[var(--success)] border border-[var(--success)]/25",
        warning: "bg-[var(--warning)]/16 text-[var(--warning)] border border-[var(--warning)]/25",
        danger: "bg-[var(--danger)]/16 text-[var(--danger)] border border-[var(--danger)]/25",
        ink: "bg-[var(--ink)] text-[var(--bg)]",
      },
    },
    defaultVariants: { tone: "neutral" },
  }
);

export function Badge({ className, tone, ...props }) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}

// Maps an invoice status → badge tone + label. Kept here so every table,
// list, and detail view renders status consistently.
export const INVOICE_STATUS = {
  draft: { tone: "neutral", labelKey: "status.draft" }, sent: { tone: "info", labelKey: "status.sent" },
  paid: { tone: "success", labelKey: "status.paid" }, overdue: { tone: "danger", labelKey: "status.overdue" },
  pending: { tone: "warning", labelKey: "status.pending" },
};

export function StatusBadge({ status, className }) {
  const { t } = useLang();
  const s = INVOICE_STATUS[status] || INVOICE_STATUS.draft;
  return (
    <Badge tone={s.tone} className={className}>
      <span className="h-1.5 w-1.5 rounded-full bg-current opacity-80" />
      {t(s.labelKey)}
    </Badge>
  );
}

export { badgeVariants };

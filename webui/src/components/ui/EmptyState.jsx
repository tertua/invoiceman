import { Card } from "@/components/ui/Card";
import { cn } from "@/lib/utils";

export function EmptyState({ icon: Icon, illustration, title, description, action, tone = "accent", className }) {
  return (
    <Card radius="lg" className={cn("flex flex-col items-center text-center py-12", className)}>
      {illustration ? (
        <div className="mb-4 text-[var(--accent)]">{illustration}</div>
      ) : Icon ? (
        <div className={cn("h-14 w-14 rounded-2xl flex items-center justify-center mb-3", tone === "neutral" ? "bg-[var(--surface-2)] text-[var(--ink-muted)] border border-[var(--border)]" : "bg-[var(--accent-soft)] text-[var(--accent-strong)]")}>
          <Icon size={22} />
        </div>
      ) : null}
      <div className="font-display text-lg font-semibold tracking-tight">{title}</div>
      {description && (
        <p className="text-sm text-[var(--ink-muted)] mt-1 max-w-sm">{description}</p>
      )}
      {action && <div className="mt-5">{action}</div>}
    </Card>
  );
}

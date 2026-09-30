import { cn } from "@/lib/utils";

export function PageHeader({ title, description, actions, className }) {
  return (
    <div className={cn("flex flex-col sm:flex-row sm:items-end sm:justify-between gap-3 sm:gap-4 mb-6", className)}>
      <div>
        <h2 className="font-display text-2xl font-semibold tracking-tight text-[var(--ink)]">
          {title}
        </h2>
        {description && (
          <p className="text-sm text-[var(--ink-muted)] mt-1">{description}</p>
        )}
      </div>
      {actions && <div className="shrink-0 w-full sm:w-auto flex flex-wrap gap-2 sm:justify-end">{actions}</div>}
    </div>
  );
}

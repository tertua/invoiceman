import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/Card";

// Extracted from AdminUsers/ProjectsTable/SettlementPanel so the three console
// tables share one dense language: compact row height, sticky header that keeps
// `--surface-2` over the page `--surface` on scroll, tabular numerics, and
// truncated monospace IDs with a tooltip.
export function AdminDenseTable({ columns, children, minWidthClassName = "min-w-[700px]", caption }) {
  return (
    <Card padding="none" className="overflow-hidden">
      <div className="overflow-x-auto">
        <table className={cn("w-full text-left", minWidthClassName)}>
          {caption && <caption className="sr-only">{caption}</caption>}
          <thead className="sticky top-0 z-10 bg-[var(--surface-2)] text-[11px] font-semibold uppercase tracking-wide text-[var(--ink-muted)]">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={cn(
                    "px-4 py-2.5",
                    column.align === "right" && "text-right",
                    column.className,
                  )}
                >
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>{children}</tbody>
        </table>
      </div>
    </Card>
  );
}

export function AdminDenseRow({ children, className }) {
  return (
    <tr
      className={cn(
        "border-t border-[var(--border)] transition-colors hover:bg-[var(--surface-2)]/60",
        className,
      )}
    >
      {children}
    </tr>
  );
}

export function AdminDenseCell({ children, className, ...props }) {
  return (
    <td className={cn("px-4 py-3 text-sm", className)} {...props}>
      {children}
    </td>
  );
}

export function AdminDenseCellMuted({ children, className, ...props }) {
  return (
    <AdminDenseCell className={cn("text-[var(--ink-muted)] tabular", className)} {...props}>
      {children}
    </AdminDenseCell>
  );
}

export function AdminDenseCellNum({ children, className, ...props }) {
  return (
    <AdminDenseCell
      className={cn("text-right tabular whitespace-nowrap font-display", className)}
      {...props}
    >
      {children}
    </AdminDenseCell>
  );
}

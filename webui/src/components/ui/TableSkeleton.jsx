import { Skeleton } from "@/components/ui/Skeleton";
import { cn } from "@/lib/utils";

// Loading placeholder that mirrors the real table layout so the loading → data
// swap does not jump.
// variant="grid": pass the SAME `grid-cols-[...]` literal the grid table uses
//   (see InvoiceTable/PaymentsTable/Expenses) plus a `columns` array of width
//   classes per cell. The caller supplies the Card shell.
// variant="table": pass `headers` (column count) + `minWidthClassName` for
//   native `<table>` pages (AdminUsers/ProjectsTable); this wraps its own Card.
export function TableSkeleton({
  variant = "grid",
  rows = 5,
  gridClassName,
  columns = [],
  headers = [],
  minWidthClassName,
  className,
}) {
  if (variant === "table") {
    const cols = headers.length || columns.length || 4;
    return (
      <div className={cn("overflow-hidden rounded-3xl border border-[var(--border)] bg-[var(--surface)]", className)}>
        <div className="overflow-x-auto">
          <table className={cn("w-full text-left", minWidthClassName)}>
            <thead className="bg-[var(--surface-2)]">
              <tr>
                {Array.from({ length: cols }).map((_, i) => (
                  <th key={i} className="px-4 py-3">
                    <Skeleton className="h-3 w-20 rounded-xl" />
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: rows }).map((_, r) => (
                <tr key={r} className="border-t border-[var(--border)]">
                  {Array.from({ length: cols }).map((_, c) => (
                    <td key={c} className="px-4 py-4">
                      <Skeleton className="h-4 w-full rounded-xl" />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }

  return (
    <div className={className}>
      <div className={cn("hidden md:grid gap-4 px-5 py-3 border-b border-[var(--border)]", gridClassName)}>
        {columns.map((w, i) => (
          <Skeleton key={i} className={cn("h-3 rounded-xl", w || "w-20")} />
        ))}
      </div>
      <div className="divide-y divide-[var(--border)]">
        {Array.from({ length: rows }).map((_, r) => (
          <div key={r} className={cn("grid grid-cols-2 gap-x-4 gap-y-1 px-5 py-4 items-center", gridClassName)}>
            {columns.map((w, i) => (
              <Skeleton key={i} className={cn("h-4 rounded-xl", w || "w-full")} />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

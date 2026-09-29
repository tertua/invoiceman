import { ArrowUpDown, Pencil, Trash2 } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { StatusBadge } from "@/components/ui/Badge";
import { useLang } from "@/context/LangContext";
import { formatMoney, formatDate, cn } from "@/lib/utils";

// Invoice list table (header + rows); row-neutral, loading/empty states stay on the page.
export function InvoiceTable({ invoices, sort, onSort, onSelect, onEdit, onDelete }) {
  const { t } = useLang();

  return (
    <Card padding="none" className="overflow-hidden">
      {/* header */}
      <div className="hidden md:grid grid-cols-[1.4fr_1.6fr_1fr_1fr_0.9fr_auto] gap-4 px-5 py-3 border-b border-[var(--border)] text-[11px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">
        <span>{t("invoices.colInvoice")}</span>
        <span>{t("invoices.colClient")}</span>
        <SortHead label={t("invoices.colIssued")} active={sort.by === "issue_date"} order={sort.order} onClick={() => onSort("issue_date")} />
        <SortHead label={t("invoices.colDue")} active={sort.by === "due_date"} order={sort.order} onClick={() => onSort("due_date")} />
        <SortHead label={t("invoices.colAmount")} active={sort.by === "total"} order={sort.order} onClick={() => onSort("total")} />
        <span className="text-right">{t("invoices.colStatus")}</span>
      </div>

      <div className="divide-y divide-[var(--border)]">
        {invoices.map((inv) => (
          <div
            key={inv.id}
            onClick={() => onSelect(inv)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.target.closest?.("button")) return;
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSelect(inv);
              }
            }}
            className="group grid grid-cols-2 md:grid-cols-[1.4fr_1.6fr_1fr_1fr_0.9fr_auto] gap-x-4 gap-y-1 px-5 py-4 cursor-pointer hover:bg-[var(--surface-2)] transition-colors items-center focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--accent)]/40"
          >
            <div className="font-semibold text-sm text-[var(--ink)] tabular">
              {inv.invoice_number}
            </div>
            <div className="text-sm text-[var(--ink)] truncate order-3 md:order-none col-span-2 md:col-span-1">
              {inv.client_name || <span className="text-[var(--ink-muted)]">{t("common.noClient")}</span>}
              {inv.client_company && (
                <span className="text-[var(--ink-muted)]"> · {inv.client_company}</span>
              )}
            </div>
            <div className="hidden md:block text-sm text-[var(--ink-muted)] tabular">
              {formatDate(inv.issue_date)}
            </div>
            <div className="hidden md:block text-sm text-[var(--ink-muted)] tabular">
              {formatDate(inv.due_date)}
            </div>
            <div className="text-sm font-semibold text-[var(--ink)] tabular text-right md:text-left">
              {formatMoney(inv.total, inv.currency)}
            </div>
            <div className="flex items-center justify-end gap-1">
              <StatusBadge status={inv.effective_status} />
              <div className="flex md:hidden md:group-hover:flex md:group-focus-within:flex items-center gap-0.5 ml-1">
                {inv.effective_status !== "paid" && inv.effective_status !== "pending" && (
                <button type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(inv);
                  }}
                  title={t("invoices.editTitle")}
                  className="h-7 w-7 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface)] hover:text-[var(--ink)]"
                >
                  <Pencil size={13} />
                </button>
                )}
                {inv.effective_status !== "paid" && inv.effective_status !== "pending" && (
                <button type="button"
                  onClick={(e) => onDelete(e, inv)}
                  title={t("invoices.deleteTitle")}
                  className="h-7 w-7 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface)] hover:text-[var(--danger)]"
                >
                  <Trash2 size={13} />
                </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function SortHead({ label, active, order, onClick }) {
  return (
    <button type="button"
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1 uppercase tracking-wider text-[11px] font-semibold hover:text-[var(--ink)] transition-colors w-fit",
        active ? "text-[var(--ink)]" : "text-[var(--ink-muted)]"
      )}
    >
      {label}
      <ArrowUpDown size={11} className={cn(active && order === "asc" && "rotate-180")} />
    </button>
  );
}

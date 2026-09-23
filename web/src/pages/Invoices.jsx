import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Plus,
  Search,
  FileText,
  Trash2,
  ArrowUpDown,
  Pencil,
} from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { SearchInput } from "@/components/ui/Input";
import { StatusBadge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { QueryError } from "@/components/ui/QueryError";
import { useLang } from "@/context/LangContext";
import { useInvoices, useDeleteInvoice } from "@/hooks/useInvoices";
import { formatMoney, formatDate, cn } from "@/lib/utils";

function getStatusTabs(t) {
  return [
    { key: "all", label: t("invoices.all") },
    { key: "draft", label: t("status.draft") },
    { key: "sent", label: t("status.sent") },
    { key: "paid", label: t("status.paid") },
    { key: "overdue", label: t("status.overdue") },
  ];
}

export default function Invoices() {
  const nav = useNavigate();
  const { t } = useLang();
  const [status, setStatus] = useState("all");
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState({ by: "issue_date", order: "desc" });

  const { data, isLoading, error } = useInvoices({
    status,
    search: search.trim() || undefined,
    sort: sort.by,
    order: sort.order,
  });
  const del = useDeleteInvoice();

  const invoices = data || [];
  const STATUS_TABS = useMemo(() => getStatusTabs(t), [t]);

  function toggleSort(by) {
    setSort((s) =>
      s.by === by ? { by, order: s.order === "asc" ? "desc" : "asc" } : { by, order: "desc" }
    );
  }

  async function onDelete(e, inv) {
    e.stopPropagation();
    if (!window.confirm(t("invoices.confirmDelete", { number: inv.invoice_number }))) return;
    await del.mutateAsync(inv.id);
  }

  return (
    <div>
      <PageHeader
        title={t("invoices.title")}
        description={t("invoices.desc")}
        actions={
          <Button variant="accent" onClick={() => nav("/invoices/new")}>
            <Plus size={16} /> {t("invoices.create")}
          </Button>
        }
      />

      <div className="flex flex-col md:flex-row md:items-center gap-3 mb-5">
        <div className="flex items-center gap-1 p-1 rounded-full bg-[var(--surface)] border border-[var(--border)] shadow-card w-fit">
          {STATUS_TABS.map((tab) => (
            <button type="button"
              key={tab.key}
              onClick={() => setStatus(tab.key)}
              className={cn(
                "h-8 px-4 rounded-full text-xs font-semibold transition-colors",
                status === tab.key
                  ? "bg-[var(--ink)] text-[var(--bg)]"
                  : "text-[var(--ink-muted)] hover:text-[var(--ink)]"
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="md:ml-auto md:w-[320px]">
          <SearchInput
            leftIcon={<Search size={16} />}
            placeholder={t("invoices.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-2xl" />
          ))}
        </div>
      ) : error ? (
        <QueryError error={error} />
      ) : invoices.length === 0 ? (
        <EmptyState
          icon={FileText}
          title={search || status !== "all" ? t("invoices.noMatching") : t("invoices.noneYet")}
          description={
            search || status !== "all"
              ? t("invoices.tryAdjusting")
              : t("invoices.createFirst")
          }
          action={
            <Button variant="accent" onClick={() => nav("/invoices/new")}>
              <Plus size={16} /> {t("invoices.create")}
            </Button>
          }
        />
      ) : (
        <Card padding="none" className="overflow-hidden">
          {/* header */}
          <div className="hidden md:grid grid-cols-[1.4fr_1.6fr_1fr_1fr_0.9fr_auto] gap-4 px-5 py-3 border-b border-[var(--border)] text-[11px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">
            <span>{t("invoices.colInvoice")}</span>
            <span>{t("invoices.colClient")}</span>
            <SortHead label={t("invoices.colIssued")} active={sort.by === "issue_date"} order={sort.order} onClick={() => toggleSort("issue_date")} />
            <SortHead label={t("invoices.colDue")} active={sort.by === "due_date"} order={sort.order} onClick={() => toggleSort("due_date")} />
            <SortHead label={t("invoices.colAmount")} active={sort.by === "total"} order={sort.order} onClick={() => toggleSort("total")} />
            <span className="text-right">{t("invoices.colStatus")}</span>
          </div>

          <div className="divide-y divide-[var(--border)]">
            {invoices.map((inv) => (
              <div
                key={inv.id}
                onClick={() => nav(`/invoices/${inv.id}`)}
                role="button"
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.target.closest?.("button")) return;
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    nav(`/invoices/${inv.id}`);
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
                    {inv.effective_status !== "paid" && (
                    <button type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        nav(`/invoices/${inv.id}/edit`);
                      }}
                      title={t("invoices.editTitle")}
                      className="h-7 w-7 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface)] hover:text-[var(--ink)]"
                    >
                      <Pencil size={13} />
                    </button>
                    )}
                    {inv.effective_status !== "paid" && (
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
      )}
    </div>
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

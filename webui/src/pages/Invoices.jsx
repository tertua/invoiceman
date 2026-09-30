import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Search, FileText } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/Button";
import { SearchInput } from "@/components/ui/Input";
import { EmptyState } from "@/components/ui/EmptyState";
import { InvoicesIllo } from "@/components/ui/EmptyIllustrations";
import { Skeleton } from "@/components/ui/Skeleton";
import { QueryError } from "@/components/ui/QueryError";
import { InvoiceTable } from "@/components/invoice/InvoiceTable";
import { InvoiceStatusTabs } from "@/components/invoice/InvoiceStatusTabs";
import { useLang } from "@/context/LangContext";
import { useInvoices, useDeleteInvoice } from "@/hooks/useInvoices";
import { useInvoiceStatusCounts } from "@/hooks/useInvoiceStatusCounts";

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
  const { data: counts, error: countsError } = useInvoiceStatusCounts();

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
        <InvoiceStatusTabs
          tabs={STATUS_TABS}
          status={status}
          onChange={setStatus}
          counts={counts}
          countsError={countsError}
          errorLabel={t("invoices.statusCountsFailed")}
        />
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
          illustration={<InvoicesIllo />}
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
        <InvoiceTable
          invoices={invoices}
          sort={sort}
          onSort={toggleSort}
          onSelect={(inv) => nav(`/invoices/${inv.id}`)}
          onEdit={(inv) => nav(`/invoices/${inv.id}/edit`)}
          onDelete={onDelete}
        />
      )}
    </div>
  );
}

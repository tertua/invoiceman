import { useState } from "react";
import { ChevronLeft, ChevronRight, Clock3, Landmark, XCircle } from "lucide-react";
import { AdminDenseCell, AdminDenseCellMuted, AdminDenseCellNum, AdminDenseRow, AdminDenseTable } from "@/components/admin/AdminDenseTable";
import { PageStat, PageStatStrip } from "@/components/admin/PageStatStrip";
import { StatusPill } from "@/components/gateway/StatusPill";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { QueryError } from "@/components/ui/QueryError";
import { useLang } from "@/context/LangContext";
import { formatDate, formatMoney } from "@/lib/utils";
import { useGatewaySettlement, useGatewayTransactionsPage } from "@/hooks/useGatewaySettlement";

function SettlementRow({ tx }) {
  const { t } = useLang();
  return (
    <AdminDenseRow>
      <AdminDenseCellMuted>{formatDate(tx.created_at)}</AdminDenseCellMuted>
      <AdminDenseCell className="max-w-[200px]">
        <code className="block truncate text-xs text-[var(--ink)]" title={[tx.order_id, tx.external_order_id, tx.provider_txn_id || tx.midtrans_txn_id].filter(Boolean).join("\n")}>{tx.order_id}</code>
      </AdminDenseCell>
      <AdminDenseCell>{tx.gateway}</AdminDenseCell>
      <AdminDenseCellMuted>{tx.payment_method || "—"}</AdminDenseCellMuted>
      <AdminDenseCellNum>{formatMoney(tx.amount_idr, "IDR")}</AdminDenseCellNum>
      <AdminDenseCell><StatusPill status={tx.status} t={t} /></AdminDenseCell>
    </AdminDenseRow>
  );
}

export function SettlementPanel() {
  const { t } = useLang();
  const [page, setPage] = useState(1);
  const { data: summaryRows = [], error: summaryError } = useGatewaySettlement();
  const { data: listData, isLoading, error: txError } = useGatewayTransactionsPage(page);
  const error = summaryError || txError;
  if (error && error.status !== 401) return <QueryError error={error} />;

  const byStatus = Object.fromEntries(summaryRows.map((row) => [row.status, row]));
  const pick = (status) => byStatus[status] || { count: 0, amount_idr: 0 };
  const success = pick("success");
  const pending = pick("pending");
  const failed = pick("failed");
  const transactions = listData?.transactions || [];
  const meta = listData?.meta || { page, total_pages: 0, total: 0 };

  return (
    <div className="space-y-4">
      <PageStatStrip>
        <PageStat
          label={t("gateway.settledTotal")}
          value={formatMoney(success.amount_idr, "IDR")}
          sub={t("gateway.countSuccess", { count: success.count })}
          icon={Landmark}
          tone="success"
        />
        <PageStat
          label={t("status.pending")}
          value={formatMoney(pending.amount_idr, "IDR")}
          sub={t("gateway.countPending", { count: pending.count })}
          icon={Clock3}
          tone="warning"
        />
        <PageStat
          label={t("gateway.failedTx")}
          value={formatMoney(failed.amount_idr, "IDR")}
          sub={t("gateway.countFailed", { count: failed.count })}
          icon={XCircle}
          tone="danger"
        />
      </PageStatStrip>

      <AdminDenseTable
        minWidthClassName="min-w-[760px]"
        columns={[
          { key: "date", label: t("gateway.date") },
          { key: "orderId", label: t("gateway.orderId") },
          { key: "gateway", label: t("gateway.gateway") },
          { key: "method", label: t("gateway.method") },
          { key: "amount", label: t("gateway.amount"), align: "right" },
          { key: "status", label: t("gateway.status") },
        ]}
      >
        {transactions.map((tx) => <SettlementRow key={tx.order_id} tx={tx} />)}
        {!isLoading && !transactions.length && (
          <AdminDenseRow>
            <td colSpan={6} className="px-4 py-8 text-center text-sm text-[var(--ink-muted)]">{t("gateway.noTransactions")}</td>
          </AdminDenseRow>
        )}
      </AdminDenseTable>

      <Card padding="none" className="flex items-center justify-between gap-3 px-4 py-3">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={page <= 1 || isLoading}
          onClick={() => setPage((current) => current - 1)}
        >
          <ChevronLeft size={14} />
          {t("gateway.prevPage")}
        </Button>
        <span className="text-xs text-[var(--ink-muted)]">
          {t("gateway.pageOf", { page: meta.page, total: meta.total_pages })}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={page >= meta.total_pages || isLoading}
          onClick={() => setPage((current) => current + 1)}
        >
          {t("gateway.nextPage")}
          <ChevronRight size={14} />
        </Button>
      </Card>
    </div>
  );
}

import { useState } from "react";
import { ChevronLeft, ChevronRight, Landmark } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { QueryError } from "@/components/ui/QueryError";
import { useLang } from "@/context/LangContext";
import { cn, formatDate, formatMoney } from "@/lib/utils";
import { useGatewaySettlement, useGatewayTransactionsPage } from "@/hooks/useGatewaySettlement";

const STATUS_TONE = {
  success: "bg-[var(--success)]/12 text-[var(--success)]",
  pending: "bg-[var(--warning)]/12 text-[var(--warning)]",
  failed: "bg-[var(--danger)]/12 text-[var(--danger)]",
  expired: "bg-[var(--surface-2)] text-[var(--ink-muted)]",
  refunded: "bg-[var(--info)]/12 text-[var(--info)]",
  partially_refunded: "bg-[var(--info)]/12 text-[var(--info)]",
};

function StatusBadge({ status }) {
  const { t } = useLang();
  const label = t(`status.${status}`);
  return (
    <span
      className={cn(
        "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold capitalize",
        STATUS_TONE[status] || "bg-[var(--surface-2)] text-[var(--ink-muted)]",
      )}
      title={label}
    >
      {label}
    </span>
  );
}

function StatCard({ label, value, sub }) {
  return (
    <Card className="min-w-[190px] flex-1">
      <div className="text-xs font-semibold uppercase tracking-wide text-[var(--ink-muted)]">{label}</div>
      <div className="mt-1 font-display text-xl font-semibold text-[var(--ink)] tabular">{value}</div>
      <div className="mt-0.5 text-xs text-[var(--ink-muted)]">{sub}</div>
    </Card>
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
      <div className="flex flex-wrap gap-4">
        <StatCard
          label={t("gateway.settledTotal")}
          value={formatMoney(success.amount_idr, "IDR")}
          sub={t("gateway.countSuccess", { count: success.count })}
        />
        <StatCard
          label={t("status.pending")}
          value={formatMoney(pending.amount_idr, "IDR")}
          sub={t("gateway.countPending", { count: pending.count })}
        />
        <StatCard
          label={t("gateway.failedTx")}
          value={formatMoney(failed.amount_idr, "IDR")}
          sub={t("gateway.countFailed", { count: failed.count })}
        />
      </div>

      <Card padding="none" className="overflow-hidden">
        <div className="flex items-center gap-2 border-b border-[var(--border)] p-4">
          <Landmark size={16} className="text-[var(--accent-strong)]" />
          <h2 className="font-display font-semibold">{t("gateway.settlement")}</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] text-left">
            <thead className="bg-[var(--surface-2)] text-xs font-semibold uppercase tracking-wide text-[var(--ink-muted)]">
              <tr>
                <th className="px-4 py-3">{t("gateway.date")}</th>
                <th className="px-4 py-3">{t("gateway.orderId")}</th>
                <th className="px-4 py-3">{t("gateway.gateway")}</th>
                <th className="px-4 py-3">{t("gateway.method")}</th>
                <th className="px-4 py-3 text-right">{t("gateway.amount")}</th>
                <th className="px-4 py-3">{t("gateway.status")}</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((tx) => (
                <tr key={tx.order_id} className="border-t border-[var(--border)]">
                  <td className="px-4 py-3 text-sm text-[var(--ink-muted)] tabular">{formatDate(tx.created_at)}</td>
                  <td className="px-4 py-3">
                    <code className="text-xs text-[var(--ink)]" title={tx.order_id}>{tx.order_id}</code>
                    {(() => {
                      const refs = [tx.external_order_id, tx.midtrans_txn_id].filter((value) => value && value !== tx.order_id);
                      if (!refs.length) return null;
                      return (
                        <div className="mt-0.5 max-w-[240px] truncate text-[11px] text-[var(--ink-muted)]" title={refs.join(" · ")}>
                          {refs.join(" · ")}
                        </div>
                      );
                    })()}
                  </td>
                  <td className="px-4 py-3 text-sm">{tx.gateway}</td>
                  <td className="px-4 py-3 text-sm text-[var(--ink-muted)]">{tx.payment_method || "—"}</td>
                  <td className="px-4 py-3 text-sm text-right tabular whitespace-nowrap">{formatMoney(tx.amount_idr, "IDR")}</td>
                  <td className="px-4 py-3"><StatusBadge status={tx.status} /></td>
                </tr>
              ))}
              {!isLoading && !transactions.length && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-sm text-[var(--ink-muted)]">{t("gateway.noTransactions")}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className="flex items-center justify-between gap-3 border-t border-[var(--border)] px-4 py-3">
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
        </div>
      </Card>
    </div>
  );
}

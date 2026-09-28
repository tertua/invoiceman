import { useState } from "react";
import { Plus, Wallet, Trash2, CreditCard } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { QueryError } from "@/components/ui/QueryError";
import { StatCard } from "@/components/dashboard/StatCard";
import { RecordPaymentModal } from "@/components/payments/RecordPaymentModal";
import { VoidPaymentModal } from "@/components/payments/VoidPaymentModal";
import { usePayments, usePaymentMutations } from "@/hooks/usePayments";
import { useLang } from "@/context/LangContext";
import { formatMoney, formatDate } from "@/lib/utils";

export default function Payments() {
  const { t } = useLang();
  const { data, isLoading, error } = usePayments();
  const { remove } = usePaymentMutations();
  const [modalOpen, setModalOpen] = useState(false);
  const [voidTarget, setVoidTarget] = useState(null);

  const payments = data?.payments || [];

  async function onVoid(reason) {
    if (!voidTarget) return;
    await remove.mutateAsync({ id: voidTarget.id, reason });
    setVoidTarget(null);
  }

  return (
    <div>
      <PageHeader
        title={t("payments.title")}
        description={t("payments.desc")}
        actions={
          <Button variant="accent" onClick={() => setModalOpen(true)}>
            <Plus size={16} /> {t("payments.record")}
          </Button>
        }
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-5 mb-6 max-w-2xl">
        <StatCard label={t("payments.totalReceived")} value={formatMoney(data?.totals?.total || 0)} icon={Wallet} accent />
        <StatCard label={t("payments.thisMonth")} value={formatMoney(data?.totals?.thisMonth || 0)} icon={CreditCard} />
      </div>

      {isLoading ? (
        <div className="space-y-2">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-2xl" />)}</div>
      ) : error ? (
        <QueryError error={error} />
      ) : payments.length === 0 ? (
        <EmptyState
          icon={Wallet}
          title={t("payments.noneYet")}
          description={t("payments.emptyDesc")}
          action={<Button variant="accent" onClick={() => setModalOpen(true)}><Plus size={16} /> {t("payments.record")}</Button>}
        />
      ) : (
        <Card padding="none" className="overflow-hidden">
          <div className="hidden md:grid grid-cols-[1fr_1.4fr_1fr_1fr_auto] gap-4 px-5 py-3 border-b border-[var(--border)] text-[11px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">
            <span>{t("payments.colDate")}</span><span>{t("payments.colInvoiceClient")}</span><span>{t("common.method")}</span><span className="text-right">{t("common.amount")}</span><span></span>
          </div>
          <div className="divide-y divide-[var(--border)]">
            {payments.map((p) => (
              <div key={p.id} className="group grid grid-cols-2 md:grid-cols-[1fr_1.4fr_1fr_1fr_auto] gap-x-4 gap-y-1 px-5 py-4 items-center">
                <div className="text-sm text-[var(--ink-muted)] tabular">{formatDate(p.paid_on)}</div>
                <div className="order-3 md:order-none col-span-2 md:col-span-1 min-w-0">
                  <div className="text-sm font-semibold text-[var(--ink)] tabular truncate">{p.invoice_number}</div>
                  <div className="text-xs text-[var(--ink-muted)] truncate">{p.client_name || t("common.noClient")}</div>
                </div>
                <div className="hidden md:block">
                  {p.method ? <Badge tone="neutral">{p.method}</Badge> : <span className="text-xs text-[var(--ink-muted)]">—</span>}
                </div>
                <div className="text-sm font-semibold text-[var(--success)] tabular text-right">{formatMoney(p.amount, p.invoice_currency)}</div>
                {p.can_void !== false ? (
                  <button type="button" onClick={() => setVoidTarget(p)} aria-label={t("payments.voidTitle")} className="col-start-2 md:col-start-auto justify-self-end h-7 w-7 rounded-full flex items-center justify-center text-[var(--ink-muted)] md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100 focus-visible:opacity-100 transition-opacity hover:bg-[var(--surface-2)] hover:text-[var(--danger)]">
                    <Trash2 size={13} />
                  </button>
                ) : (
                  <span aria-hidden="true" className="col-start-2 md:col-start-auto justify-self-end h-7 w-7" />
                )}
              </div>
            ))}
          </div>
        </Card>
      )}

      <RecordPaymentModal open={modalOpen} onClose={() => setModalOpen(false)} />
      <VoidPaymentModal
        open={!!voidTarget}
        onClose={() => setVoidTarget(null)}
        payment={voidTarget}
        currency={voidTarget?.invoice_currency}
        onVoid={onVoid}
      />
    </div>
  );
}

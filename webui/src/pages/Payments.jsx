import { useState } from "react";
import { Plus, Wallet, CreditCard } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { QueryError } from "@/components/ui/QueryError";
import { StatCard } from "@/components/dashboard/StatCard";
import { RecordPaymentModal } from "@/components/payments/RecordPaymentModal";
import { VoidPaymentModal } from "@/components/payments/VoidPaymentModal";
import { PaymentsTable } from "@/components/payments/PaymentsTable";
import { usePayments, usePaymentMutations } from "@/hooks/usePayments";
import { useLang } from "@/context/LangContext";
import { formatMoney } from "@/lib/utils";

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
        <PaymentsTable payments={payments} onVoid={setVoidTarget} />
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

import { Trash2 } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { useLang } from "@/context/LangContext";
import { formatMoney, formatDate } from "@/lib/utils";
import { paymentMethodLabel } from "@/lib/paymentLabels";

export function PaymentsTable({ payments, onVoid }) {
  const { t } = useLang();
  return (
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
              {p.method ? <Badge tone="neutral">{paymentMethodLabel(p.method)}</Badge> : <span className="text-xs text-[var(--ink-muted)]">—</span>}
            </div>
            <div className="text-sm font-semibold text-[var(--success)] tabular text-right">{formatMoney(p.amount, p.invoice_currency)}</div>
            {p.can_void !== false ? (
              <button type="button" onClick={() => onVoid(p)} aria-label={t("payments.voidTitle")} className="col-start-2 md:col-start-auto justify-self-end h-7 w-7 rounded-full flex items-center justify-center text-[var(--ink-muted)] md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100 focus-visible:opacity-100 transition-opacity hover:bg-[var(--surface-2)] hover:text-[var(--danger)]">
                <Trash2 size={13} />
              </button>
            ) : (
              <span aria-hidden="true" className="col-start-2 md:col-start-auto justify-self-end h-7 w-7" />
            )}
          </div>
        ))}
      </div>
    </Card>
  );
}

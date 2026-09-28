import { CheckCircle2 } from "lucide-react";
import { t } from "@/lib/i18n";

// InvoiceHead is the top strip of the public pay card: who the bill is for,
// the invoice number, and the paid / pending / unpaid pill.
export default function InvoiceHead({ invoice, isPaid, isPending, lang }) {
  return (
    <div className="flex items-start justify-between gap-4 pb-5 border-b border-[var(--border)]">
      <div>
        <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">
          {t(lang, "public.billTo")}
        </div>
        <div className="text-sm font-semibold text-[var(--ink)]">{invoice.client_name || "—"}</div>
        {invoice.client_company && <div className="text-xs text-[var(--ink-muted)]">{invoice.client_company}</div>}
      </div>
      <div className="text-right">
        <div className="font-display text-xl font-bold tracking-wide text-[var(--accent-strong)]">
          {t(lang, "common.invoice").toUpperCase()}
        </div>
        <div className="text-sm text-[var(--ink-muted)] mt-0.5 tabular">{invoice.invoice_number}</div>
        <div className={`mt-2 inline-flex items-center gap-1 text-[11px] font-semibold px-2.5 py-1 rounded-full ${isPaid ? "bg-[var(--success)]/12 text-[var(--success)]" : isPending ? "bg-[var(--warning)]/12 text-[var(--warning)]" : "bg-[var(--danger)]/10 text-[var(--danger)]"}`}>
          {isPaid ? <CheckCircle2 size={12} /> : null}
          {isPaid ? t(lang, "public.paid") : isPending ? t(lang, "status.pending") : t(lang, "public.unpaid")}
        </div>
      </div>
    </div>
  );
}

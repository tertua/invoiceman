import { Card } from "@/components/ui/Card";
import { useLang } from "@/context/LangContext";
import { formatMoney, formatDate } from "@/lib/utils";

export function InvoicePreview({ invoice, settings }) {
  const { t } = useLang();
  const s = settings || {};
  const currency = invoice.currency;
  return (
    <Card padding="lg">
      <div className="flex items-start justify-between gap-4 pb-6 border-b border-[var(--border)]">
        <div>
          {s.logo_url ? (
            <img src={s.logo_url} alt="" className="h-12 w-12 object-contain mb-2 rounded" />
          ) : null}
          <div className="font-display text-lg font-semibold text-[var(--ink)]">
            {s.company_name || "Your Company"}
          </div>
          {s.address && <div className="text-xs text-[var(--ink-muted)] max-w-[220px]">{s.address}</div>}
          {s.email && <div className="text-xs text-[var(--ink-muted)]">{s.email}</div>}
        </div>
        <div className="text-right">
          <div className="font-display text-2xl font-bold tracking-wide text-[var(--accent-strong)]">
            {t("common.invoice").toUpperCase()}
          </div>
          <div className="text-sm text-[var(--ink-muted)] mt-1 tabular">{invoice.invoice_number}</div>
        </div>
      </div>

      <div className="flex items-start justify-between gap-4 py-6">
        <div>
          <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">
            {t("invDetail.billTo")}
          </div>
          <div className="text-sm font-semibold text-[var(--ink)]">{invoice.client_name || "—"}</div>
          {invoice.client_company && <div className="text-xs text-[var(--ink-muted)]">{invoice.client_company}</div>}
          {invoice.client_email && <div className="text-xs text-[var(--ink-muted)]">{invoice.client_email}</div>}
        </div>
        <div className="text-right text-sm space-y-1">
          <MetaLine label={t("invDetail.issued")} value={formatDate(invoice.issue_date)} />
          <MetaLine label={t("invDetail.due")} value={formatDate(invoice.due_date)} />
        </div>
      </div>

      {/* items */}
      <div className="grid grid-cols-[1fr_60px_90px_90px] gap-3 pb-2 border-b border-[var(--ink)] text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">
        <span>{t("common.description")}</span>
        <span className="text-right">{t("common.qty")}</span>
        <span className="text-right">{t("common.rate")}</span>
        <span className="text-right">{t("common.amount")}</span>
      </div>
      {(invoice.items || []).map((it, i) => (
        <div key={i} className="grid grid-cols-[1fr_60px_90px_90px] gap-3 py-2.5 border-b border-[var(--border)] text-sm">
          <span className="text-[var(--ink)]">{it.description || "—"}</span>
          <span className="text-right tabular text-[var(--ink-muted)]">{Number(it.quantity)}</span>
          <span className="text-right tabular text-[var(--ink-muted)]">{formatMoney(it.rate, currency)}</span>
          <span className="text-right tabular text-[var(--ink)] font-medium">{formatMoney(it.amount, currency)}</span>
        </div>
      ))}

      {/* totals */}
      <div className="ml-auto w-full max-w-[260px] mt-5 space-y-2 text-sm">
        <TotalLine label={t("common.subtotal")} value={formatMoney(invoice.subtotal, currency)} />
        {Number(invoice.discount) > 0 && (
          <TotalLine label={t("common.discount")} value={`− ${formatMoney(invoice.discount, currency)}`} />
        )}
        <TotalLine label={t("invDetail.taxLine", { n: Number(invoice.tax_rate) })} value={formatMoney(invoice.tax_amount, currency)} />
        <div className="flex items-center justify-between pt-3 border-t border-[var(--ink)]">
          <span className="font-display font-semibold">{t("common.total")}</span>
          <span className="font-display text-xl font-semibold tabular text-[var(--accent-strong)]">
            {formatMoney(invoice.total, currency)}
          </span>
        </div>
      </div>

      {(invoice.notes || invoice.terms) && (
        <div className="mt-8 pt-5 border-t border-[var(--border)] space-y-3">
          {invoice.notes && (
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">{t("common.notes")}</div>
              <p className="text-sm text-[var(--ink)] whitespace-pre-line">{invoice.notes}</p>
            </div>
          )}
          {invoice.terms && (
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">{t("common.terms")}</div>
              <p className="text-sm text-[var(--ink)] whitespace-pre-line">{invoice.terms}</p>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}

function MetaLine({ label, value }) {
  return (
    <div className="flex items-center justify-end gap-3">
      <span className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">{label}</span>
      <span className="tabular text-[var(--ink)] w-24 text-right">{value}</span>
    </div>
  );
}

export function TotalLine({ label, value }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-[var(--ink-muted)]">{label}</span>
      <span className="tabular text-[var(--ink)]">{value}</span>
    </div>
  );
}

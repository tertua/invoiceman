import { t } from "@/lib/i18n";
import { formatMoney, formatDate } from "@/lib/utils";

export function Row({ label, value, bold }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-[var(--ink-muted)]">{label}</span>
      <span className={`text-sm tabular ${bold ? "font-semibold text-[var(--ink)]" : "text-[var(--ink)]"}`}>{value}</span>
    </div>
  );
}

// Line items + totals block of the public invoice. Extracted from PublicPay so
// the page stays a composer and the step indicator/countdown can be added
// without growing it past its baseline.
export default function InvoiceBreakdown({ invoice, cur, lang }) {
  return (
    <>
      <div className="mt-4 space-y-2">
        {(invoice.items || []).map((it, i) => (
          <div key={i} className="flex items-start justify-between gap-3 text-sm">
            <div className="min-w-0 flex-1">
              <div className="text-[var(--ink)]">{it.description || "—"}</div>
              <div className="text-xs text-[var(--ink-muted)]">{Number(it.quantity)} × {formatMoney(it.rate, cur)}</div>
            </div>
            <div className="tabular text-[var(--ink)] font-medium">{formatMoney(it.amount, cur)}</div>
          </div>
        ))}
      </div>

      <div className="mt-4 pt-4 border-t border-[var(--border)] space-y-1.5">
        <Row label={t(lang, "common.subtotal")} value={formatMoney(invoice.subtotal, cur)} />
        {Number(invoice.discount) > 0 && <Row label={t(lang, "common.discount")} value={`− ${formatMoney(invoice.discount, cur)}`} />}
        <Row label={t(lang, "invDetail.taxLine", { n: Number(invoice.tax_rate) })} value={formatMoney(invoice.tax_amount, cur)} />
        <Row label={t(lang, "invDetail.due")} value={formatDate(invoice.due_date)} />
        <div className="flex items-center justify-between pt-2 border-t border-[var(--ink)]">
          <span className="font-display font-semibold">{t(lang, "common.total")}</span>
          <span className="font-display text-xl font-semibold tabular text-[var(--accent-strong)]">
            {formatMoney(invoice.total, cur)}
          </span>
        </div>
      </div>
    </>
  );
}

import { useCallback, useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2 } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { InvoicePdfDownload } from "@/components/invoice/InvoicePdfDownload";
import PublicShell from "@/components/publicpay/PublicShell";
import PayPanel from "@/components/publicpay/PayPanel";
import InvoiceHead from "@/components/publicpay/InvoiceHead";
import { PublicErrorCard } from "@/components/publicpay/PublicErrorCard";
import { publicPayApi } from "@/api/publicPay";
import { t } from "@/lib/i18n";
import { formatMoney, formatDate, setLocale } from "@/lib/utils";

const LANG_STORAGE_KEY = "arr-lang";

function detectLang(searchParams) {
  const param = searchParams?.get("lang");
  if (param === "en" || param === "id") return param;
  if (typeof window !== "undefined") {
    const stored = localStorage.getItem(LANG_STORAGE_KEY);
    if (stored === "en" || stored === "id") return stored;
  }
  if (typeof navigator !== "undefined" && navigator.language?.toLowerCase().startsWith("id")) return "id";
  return "en";
}

function Row({ label, value, bold }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-[var(--ink-muted)]">{label}</span>
      <span className={`text-sm tabular ${bold ? "font-semibold text-[var(--ink)]" : "text-[var(--ink)]"}`}>{value}</span>
    </div>
  );
}

// Dev path — public pay page (orchestrator scaffold)
//   [done]  PayPanel swaps in the crypto/QRIS widget or the MethodPicker
//   [next]  keep MethodPicker visible under the widget so the payer can switch
//           method without a reload
//   [later] order methods by audience (domestic IDR first)
// Seam: data contract from publicPayApi.get plus PayPanel's props — the page
// only composes, it never talks to a gateway directly.
// End dev path
export default function PublicPay() {
  const { token } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const [lang, setLang] = useState(() => detectLang(searchParams));
  const [data, setData] = useState(null);
  const [err, setErr] = useState("");

  const refresh = useCallback(() => {
    publicPayApi.get(token).then(setData).catch(() => {});
  }, [token]);

  useEffect(() => {
    document.documentElement.setAttribute("lang", lang);
    setLocale(lang);
  }, [lang]);

  function changeLang(next) {
    if (next !== "en" && next !== "id") return;
    setLang(next);
    try {
      localStorage.setItem(LANG_STORAGE_KEY, next);
    } catch {
      // Private mode may block storage; the ?lang= param still applies.
    }
    setSearchParams(
      (prev) => {
        const nextParams = new URLSearchParams(prev);
        nextParams.set("lang", next);
        return nextParams;
      },
      { replace: true }
    );
  }

  useEffect(() => {
    let cancelled = false;
    publicPayApi
      .get(token)
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch((e) => {
        if (!cancelled) setErr({ status: e?.status, message: e.message || t(lang, "public.invalidLink") });
      });
    return () => {
      cancelled = true;
    };
  }, [token, lang]);

  useEffect(() => {
    if (!data || (data.can_pay === false && data.invoice?.effective_status !== "pending")) return undefined;
    let cancelled = false, delay = 5000, timer;
    const poll = async () => {
      try {
        const status = await publicPayApi.status(token);
        if (cancelled) return;
        delay = 5000;
        if (status.status === "paid") { const r = await publicPayApi.get(token); if (!cancelled) setData(r); }
      } catch {
        delay = Math.min(delay * 2, 30000); // back off while the status call keeps failing
      }
      if (!cancelled) timer = window.setTimeout(poll, delay);
    };
    timer = window.setTimeout(poll, delay);
    return () => { cancelled = true; window.clearTimeout(timer); };
  }, [data, token]);

  if (err) {
    return (
      <PublicShell lang={lang} onLang={changeLang}>
        <PublicErrorCard lang={lang} status={err.status} message={err.message} />
      </PublicShell>
    );
  }
  if (!data) {
    return (
      <PublicShell lang={lang} onLang={changeLang}>
        <Loader2 className="animate-spin text-[var(--ink-muted)]" size={22} />
      </PublicShell>
    );
  }

  const { invoice, branding, can_pay, methods = [] } = data;
  const cur = invoice.currency || "IDR";
  const isPaid = invoice.effective_status === "paid", isPending = invoice.effective_status === "pending";

  return (
    <PublicShell branding={branding} lang={lang} onLang={changeLang}>
      <Card padding="lg" className="w-full max-w-[520px]">
        <InvoiceHead invoice={invoice} isPaid={isPaid} isPending={isPending} lang={lang} />

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

        {isPaid ? (
          <div className="mt-6 rounded-2xl border border-[var(--success)]/25 bg-[var(--success)]/8 p-4">
            <div className="flex items-center gap-2 text-sm font-semibold text-[var(--success)] mb-2">
              <CheckCircle2 size={16} /> {t(lang, "public.thanksPaid")}
            </div>
            <div className="space-y-1.5">
              {(invoice.payments || []).map((p) => (
                <Row key={p.id} label={`${formatDate(p.paid_on)} · ${p.method || "Online"}`} value={formatMoney(p.amount, cur)} bold />
              ))}
            </div>
            <div className="mt-4">
              <InvoicePdfDownload
                invoice={invoice}
                settings={branding}
                lang={lang}
                publicView
                label={t(lang, "public.downloadPdf")}
              />
            </div>
          </div>
        ) : can_pay ? (
          <PayPanel token={token} methods={methods} lang={lang} gateway={data.gateway} onRefresh={refresh} />
        ) : (
          <div className="mt-6 text-center text-sm text-[var(--ink-muted)]">{t(lang, "public.notPayable")}</div>
        )}
      </Card>
    </PublicShell>
  );
}

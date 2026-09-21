import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { CheckCircle2, Loader2, ArrowRight, ShieldCheck } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { InvoicePdfDownload } from "@/components/invoice/InvoicePdfDownload";
import { publicPayApi } from "@/api/publicPay";
import { loadMidtransSnap } from "@/lib/midtrans";
import { t } from "@/lib/i18n";
import { formatMoney, formatDate, setLocale } from "@/lib/utils";

function detectLang() {
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

export default function PublicPay() {
  const { token } = useParams();
  const lang = detectLang();
  const [data, setData] = useState(null);
  const [err, setErr] = useState("");
  const [paying, setPaying] = useState(false);
  const [payErr, setPayErr] = useState("");

  useEffect(() => {
    setLocale(lang);
  }, [lang]);

  useEffect(() => {
    let cancelled = false;
    publicPayApi
      .get(token)
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch((e) => {
        if (!cancelled) setErr(e.message || t(lang, "public.invalidLink"));
      });
    return () => {
      cancelled = true;
    };
  }, [token, lang]);

  useEffect(() => {
    if (!data || data.invoice?.effective_status === "paid") return undefined;
    let cancelled = false;
    const poll = async () => {
      try {
        const status = await publicPayApi.status(token);
        if (cancelled || status.status !== "paid") return;
        const refreshed = await publicPayApi.get(token);
        if (!cancelled) setData(refreshed);
      } catch {
        // Payment status is best-effort; the page remains usable if polling fails.
      }
    };
    const interval = window.setInterval(poll, 5000);
    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [data, token]);

  async function pay() {
    setPaying(true);
    setPayErr("");
    try {
      const res = await publicPayApi.createTransaction(token);
      if (res.snap_token && data.gateway?.client_key) {
        const snap = await loadMidtransSnap(data.gateway.is_production);
        snap.pay(res.snap_token, {
          onClose: () => setPaying(false),
          onError: () => setPaying(false),
          onSuccess: () => window.location.reload(),
        });
        return;
      }
      if (res.redirect_url) window.location.href = res.redirect_url;
    } catch (e) {
      setPayErr(e.message || t(lang, "public.notPayable"));
      setPaying(false);
    }
  }

  if (err) {
    return (
      <PublicShell>
        <Card padding="lg" className="max-w-md w-full text-center">
          <p className="text-sm text-[var(--ink-muted)]">{err}</p>
        </Card>
      </PublicShell>
    );
  }
  if (!data) {
    return (
      <PublicShell>
        <Loader2 className="animate-spin text-[var(--ink-muted)]" size={22} />
      </PublicShell>
    );
  }

  const { invoice, branding, can_pay } = data;
  const cur = invoice.currency || "IDR";
  const isPaid = invoice.effective_status === "paid";

  return (
    <PublicShell branding={branding}>
      <Card padding="lg" className="w-full max-w-[520px]">
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
            <div className={`mt-2 inline-flex items-center gap-1 text-[11px] font-semibold px-2.5 py-1 rounded-full ${isPaid ? "bg-[var(--success)]/12 text-[var(--success)]" : "bg-[var(--danger)]/10 text-[var(--danger)]"}`}>
              {isPaid ? <CheckCircle2 size={12} /> : null}
              {isPaid ? t(lang, "public.paid") : t(lang, "public.unpaid")}
            </div>
          </div>
        </div>

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
          <div className="mt-6">
            <Button variant="accent" className="w-full" onClick={pay} disabled={paying}>
              {paying ? <Loader2 size={15} className="animate-spin" /> : <ArrowRight size={15} />}
              {paying ? t(lang, "public.paying") : t(lang, "public.payNow")}
            </Button>
            {payErr && <p className="text-xs text-[var(--danger)] mt-3 text-center">{payErr}</p>}
            <div className="flex items-center justify-center gap-1.5 mt-4 text-[11px] text-[var(--ink-muted)]">
              <ShieldCheck size={13} /> {t(lang, "public.secureBy")}
            </div>
          </div>
        ) : (
          <div className="mt-6 text-center text-sm text-[var(--ink-muted)]">{t(lang, "public.notPayable")}</div>
        )}
      </Card>
    </PublicShell>
  );
}

function PublicShell({ children, branding }) {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-6 p-6 bg-[var(--bg)]">
      {branding?.logo_url ? (
        <img src={branding.logo_url} alt={branding.company_name || "logo"} className="h-12 w-12 object-contain rounded" />
      ) : null}
      {branding?.company_name ? (
        <div className="font-display text-lg font-semibold text-[var(--ink)]">{branding.company_name}</div>
      ) : null}
      {children}
      <p className="text-[11px] text-[var(--ink-muted)]">
        Powered by{" "}
        <Link to="/" className="font-semibold text-[var(--ink)] hover:underline">
          Invoicer
        </Link>
      </p>
    </div>
  );
}

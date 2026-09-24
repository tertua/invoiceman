import { useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2, ArrowRight, ShieldCheck } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { InvoicePdfDownload } from "@/components/invoice/InvoicePdfDownload";
import PublicShell from "@/components/publicpay/PublicShell";
import { publicPayApi } from "@/api/publicPay";
import { loadMidtransSnap } from "@/lib/midtrans";
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

// MethodPicker lets the payer choose how to pay; the gateway intent (and its
// paylink) is only created after a method is chosen.
function MethodPicker({ methods, lang, onPick, pending, error }) {
  return (
    <div className="mt-6">
      <div className="text-xs font-semibold uppercase tracking-wider text-[var(--ink-muted)] mb-3">
        {t(lang, "public.chooseMethod")}
      </div>
      {methods.length ? (
        <div className="space-y-2">
          {methods.map((m) => (
            <button
              key={m.id}
              type="button"
              onClick={() => onPick(m.id)}
              disabled={!!pending}
              className="w-full flex items-center justify-between gap-3 rounded-2xl border border-[var(--border)] bg-[var(--surface)] px-4 py-3 text-left transition-colors hover:border-[var(--accent)]/50 hover:bg-[var(--surface-2)] disabled:opacity-60"
            >
              <span className="text-sm font-semibold text-[var(--ink)]">{m.name}</span>
              <span className="flex items-center gap-2">
                <span className="text-sm tabular text-[var(--ink)]">{formatMoney(m.amount, m.currency)}</span>
                {pending === m.id ? (
                  <Loader2 size={15} className="animate-spin text-[var(--accent-strong)]" />
                ) : (
                  <ArrowRight size={15} className="text-[var(--accent-strong)]" />
                )}
              </span>
            </button>
          ))}
        </div>
      ) : (
        <p className="text-sm text-[var(--ink-muted)]">{t(lang, "public.notPayable")}</p>
      )}
      {error && methods.length ? <p className="text-xs text-[var(--danger)] mt-3 text-center">{error}</p> : null}
      <div className="flex items-center justify-center gap-1.5 mt-4 text-[11px] text-[var(--ink-muted)]">
        <ShieldCheck size={13} /> {t(lang, "public.secureBy")}
      </div>
    </div>
  );
}

export default function PublicPay() {
  const { token } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const [lang, setLang] = useState(() => detectLang(searchParams));
  const [data, setData] = useState(null);
  const [err, setErr] = useState("");
  const [pending, setPending] = useState("");
  const [payErr, setPayErr] = useState("");

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

  async function pay(method) {
    setPending(method);
    setPayErr("");
    try {
      const res = await publicPayApi.createTransaction(token, method);
      if (res.snap_token && data.gateway?.client_key) {
        const snap = await loadMidtransSnap(data.gateway.is_production);
        snap.pay(res.snap_token, {
          onClose: () => setPending(""),
          onError: () => setPending(""),
          onSuccess: () => window.location.reload(),
        });
        return;
      }
      const hosted = res.redirect_url || res.payment_url;
      if (hosted) {
        window.location.href = hosted;
        return;
      }
      setPending("");
    } catch (e) {
      setPayErr(e.message || t(lang, "public.notPayable"));
      setPending("");
    }
  }

  if (err) {
    return (
      <PublicShell lang={lang} onLang={changeLang}>
        <Card padding="lg" className="max-w-md w-full text-center">
          <p className="text-sm text-[var(--ink-muted)]">{err}</p>
        </Card>
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
          <MethodPicker methods={methods} lang={lang} onPick={pay} pending={pending} error={payErr} />
        ) : (
          <div className="mt-6 text-center text-sm text-[var(--ink-muted)]">{t(lang, "public.notPayable")}</div>
        )}
      </Card>
    </PublicShell>
  );
}

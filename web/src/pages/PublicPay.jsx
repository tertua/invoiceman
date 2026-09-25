import { useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2 } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { InvoicePdfDownload } from "@/components/invoice/InvoicePdfDownload";
import PublicShell from "@/components/publicpay/PublicShell";
import CryptoWidget from "@/components/publicpay/CryptoWidget";
import MethodPicker from "@/components/publicpay/MethodPicker";
import InvoiceHead from "@/components/publicpay/InvoiceHead";
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

// Dev path — public pay page (orchestrator scaffold)
//   [done]  picking "crypto" swaps in CryptoWidget and hides the MethodPicker
//   [next]  keep MethodPicker visible under the widget so the payer can switch
//           method without a reload
//   [later] order methods by audience (domestic IDR first)
// Seam: data contract from publicPayApi.get plus the picker/widget props —
// the page only composes, it never talks to a gateway directly.
// End dev path
export default function PublicPay() {
  const { token } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const [lang, setLang] = useState(() => detectLang(searchParams));
  const [data, setData] = useState(null);
  const [err, setErr] = useState("");
  const [pending, setPending] = useState("");
  const [payErr, setPayErr] = useState("");
  const [cryptoActive, setCryptoActive] = useState(false);

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
      if (method === "crypto") {
        setCryptoActive(true);
        setPending("");
        return;
      }
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
          cryptoActive ? (
            <CryptoWidget
              key={token}
              token={token}
              lang={lang}
              onError={(m) => {
                setPayErr(m);
                setCryptoActive(false);
              }}
              onPaid={() => {
                setCryptoActive(false);
                publicPayApi.get(token).then(setData).catch(() => {});
              }}
            />
          ) : (
            <MethodPicker methods={methods} lang={lang} onPick={pay} pending={pending} error={payErr} />
          )
        ) : (
          <div className="mt-6 text-center text-sm text-[var(--ink-muted)]">{t(lang, "public.notPayable")}</div>
        )}
      </Card>
    </PublicShell>
  );
}

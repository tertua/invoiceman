import { useCallback, useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2 } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { InvoicePdfDownload } from "@/components/invoice/InvoicePdfDownload";
import PublicShell from "@/components/publicpay/PublicShell";
import PayPanel from "@/components/publicpay/PayPanel";
import InvoiceHead from "@/components/publicpay/InvoiceHead";
import InvoiceBreakdown, { Row } from "@/components/publicpay/InvoiceBreakdown";
import StepIndicator from "@/components/publicpay/StepIndicator";
import { PublicErrorCard } from "@/components/publicpay/PublicErrorCard";
import { publicPayApi } from "@/api/publicPay";
import { t } from "@/lib/i18n";
import { resolvePayStep } from "@/lib/payStep";
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
  const [panelStep, setPanelStep] = useState("choose");

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
    const live = data.invoice?.effective_status === "pending";
    let cancelled = false, timer, delay = live ? 2500 : 10000;
    const poll = async () => {
      try {
        if (!document.hidden) {
          const status = await publicPayApi.status(token);
          if (cancelled) return;
          if (status.status === "paid") { const r = await publicPayApi.get(token); if (!cancelled) setData(r); }
        }
        delay = live ? 2500 : 10000;
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
  const step = resolvePayStep({ isPaid, canPay: can_pay, panelStep });

  return (
    <PublicShell branding={branding} lang={lang} onLang={changeLang}>
      <Card padding="lg" className="w-full max-w-[520px]">
        <div className="mb-4 flex justify-center">
          <StepIndicator current={step} lang={lang} />
        </div>
        <InvoiceHead invoice={invoice} isPaid={isPaid} isPending={isPending} lang={lang} />

        <InvoiceBreakdown invoice={invoice} cur={cur} lang={lang} />

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
          <PayPanel token={token} methods={methods} lang={lang} gateway={data.gateway} onRefresh={refresh} onStepChange={setPanelStep} />
        ) : (
          <div className="mt-6 text-center text-sm text-[var(--ink-muted)]">{t(lang, "public.notPayable")}</div>
        )}
      </Card>
    </PublicShell>
  );
}

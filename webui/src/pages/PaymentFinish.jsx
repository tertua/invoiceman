import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { motion, useReducedMotion } from "framer-motion";
import { CheckCircle2, Clock, XCircle } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import PublicShell from "@/components/publicpay/PublicShell";
import { t } from "@/lib/i18n";
import { setLocale } from "@/lib/utils";
import { finishState } from "@/lib/paymentFinish";

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

const STATE_META = {
  success: { icon: CheckCircle2, tone: "var(--success)", titleKey: "finish.success", descKey: "finish.successDesc" },
  pending: { icon: Clock, tone: "var(--warning)", titleKey: "finish.pending", descKey: "finish.pendingDesc" },
  failed: { icon: XCircle, tone: "var(--danger)", titleKey: "finish.failed", descKey: "finish.failedDesc" },
  unknown: { icon: Clock, tone: "var(--ink-muted)", titleKey: "finish.title", descKey: "finish.unknownDesc" },
};

// Static thank-you page for the browser checkout Finish Redirect URL.
// Register `{ORIGIN}/payment/finish` in the provider dashboard; the widget
// appends `?order_id=&status_code=&transaction_status=`. No backend call: the
// page only reflects the redirect params, settlement truth stays on webhooks.
export default function PaymentFinish() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [lang, setLang] = useState(() => detectLang(searchParams));
  const reduce = useReducedMotion();
  const orderId = searchParams.get("order_id") || "";
  const meta = STATE_META[finishState(searchParams.get("transaction_status"))];
  const Icon = meta.icon;

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

  return (
    <PublicShell lang={lang} onLang={changeLang}>
      <Card padding="lg" className="max-w-md w-full flex flex-col items-center text-center py-12">
        <motion.div
          className="h-14 w-14 rounded-2xl flex items-center justify-center mb-4"
          role="img"
          aria-label={t(lang, "finish.successAnimAria")}
          style={{ backgroundColor: `color-mix(in srgb, ${meta.tone} 12%, transparent)`, color: meta.tone }}
          initial={reduce ? false : { scale: 0.6, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={reduce ? { duration: 0 } : { type: "spring", stiffness: 260, damping: 18 }}
        >
          <Icon size={26} />
        </motion.div>
        <div className="font-display text-lg font-semibold tracking-tight text-[var(--ink)]">
          {t(lang, meta.titleKey)}
        </div>
        <p className="text-sm text-[var(--ink-muted)] mt-1.5 max-w-sm">{t(lang, meta.descKey)}</p>
        {orderId ? (
          <div className="mt-4 w-full flex items-center justify-between gap-4 rounded-xl border border-[var(--border)] px-4 py-2.5 text-left">
            <span className="text-sm text-[var(--ink-muted)]">{t(lang, "finish.orderLabel")}</span>
            <span className="text-sm font-mono tabular text-[var(--ink)] break-all">{orderId}</span>
          </div>
        ) : null}
        <div className="mt-6">
          <Link to="/">
            <Button variant="outline" size="sm">{t(lang, "public.backHome")}</Button>
          </Link>
        </div>
      </Card>
    </PublicShell>
  );
}

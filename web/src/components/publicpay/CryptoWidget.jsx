import { useEffect, useRef, useState } from "react";
import { Loader2, Copy, Check, Download } from "lucide-react";
import { createPublicTransaction, newIntentKey } from "@/api/publicIntent";
import { downloadImage } from "@/lib/download";
import { t } from "@/lib/i18n";

const DEFAULT_TIMEOUT_SECONDS = 910; // ~NOWPayments default payment timeout
const DEFAULT_PAY_CURRENCY = "usdttrc20";
const COPIED_FEEDBACK_DURATION = 2000; // ms to show "copied" feedback
const QR_SIZE = 200;
const QR_MARGIN = 1;

function formatCountdown(left) {
  const m = String(Math.floor(left / 60)).padStart(2, "0");
  const s = String(left % 60).padStart(2, "0");
  return `${m}:${s}`;
}

function payErrorMessage(e, lang) {
  const code = e?.details?.code;
  if (code === "amount_below_minimum") return t(lang, "public.payAmountMinimum");
  if (code === "rate_limited" || e?.status === 429) return t(lang, "public.payRateLimited");
  return t(lang, "public.payCreateFailed");
}

function currencyLabel(payCurrency) {
  const c = (payCurrency || "").toLowerCase();
  if (c.startsWith("usdt")) {
    const network = c.slice(4);
    return network ? `USDT (${network.toUpperCase()})` : "USDT";
  }
  return (payCurrency || "USDT").toUpperCase();
}

// Dev path — on-page crypto widget (scaffold, one network on purpose)
//   [done]  USDT TRC20 direct payment: QR, address, amount, countdown,
//           retry on 429, localized failure messages
//   [next]  DEFAULT_PAY_CURRENCY stays the single swap point for the pay currency
//   [later] network picker (ERC20, BEP20) once the widget settles
// Seam: props {token, lang, onError} + createPublicTransaction(token,
// "crypto", extra, idempotencyKey) — one UUID per widget, stable across
// StrictMode double-effects and language toggles so a re-run replays
// instead of double-charging.
// End dev path
export default function CryptoWidget({ token, lang, onError }) {
  const [intent, setIntent] = useState(null);
  const [pending, setPending] = useState(true);
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef(null);
  const keysRef = useRef({});
  const [left, setLeft] = useState(DEFAULT_TIMEOUT_SECONDS);
  const [qrSrc, setQrSrc] = useState("");
  const [saving, setSaving] = useState(false);

  function qrFileName() {
    const id = String(intent?.payment_id || intent?.order_id || "").replace(/[^A-Za-z0-9-_]/g, "").slice(0, 48);
    return `${id || "crypto-qr"}.png`;
  }

  async function saveQr() {
    if (!qrSrc || saving) return;
    setSaving(true);
    const ok = await downloadImage(qrSrc, qrFileName());
    setSaving(false);
    if (!ok) window.open(qrSrc, "_blank", "noopener");
  }

  useEffect(() => {
    let cancelled = false;
    setPending(true);
    if (!keysRef.current.key) keysRef.current.key = newIntentKey(0);
    createPublicTransaction(token, "crypto", { pay_currency: DEFAULT_PAY_CURRENCY }, keysRef.current.key)
      .then((res) => {
        if (cancelled) return;
        if (res.address) {
          setIntent(res);
        } else if (res.payment_url) {
          window.location.href = res.payment_url;
        } else {
          onError(t(lang, "public.payNoAddress"));
        }
      })
      .catch((e) => {
        if (!cancelled) onError(payErrorMessage(e, lang));
      })
      .finally(() => {
        if (!cancelled) setPending(false);
      });
    return () => {
      cancelled = true;
      if (copyTimer.current) clearTimeout(copyTimer.current);
    };
  }, [token, lang, onError]);

  useEffect(() => {
    let mounted = true;
    if (intent?.address) {
      import("qrcode").then(({ default: QRCode }) => {
        if (!mounted) return;
        QRCode.toDataURL(intent.address, { margin: QR_MARGIN, width: QR_SIZE })
          .then(setQrSrc)
          .catch((err) => console.warn("QR generation failed:", err));
      });
    }
    return () => {
      mounted = false;
    };
  }, [intent]);

  useEffect(() => {
    const start = Date.now();
    const total = DEFAULT_TIMEOUT_SECONDS * 1000;
    const timer = setInterval(() => {
      const remaining = Math.max(0, Math.round((start + total - Date.now()) / 1000));
      setLeft(remaining);
      if (remaining <= 0) {
        clearInterval(timer);
      }
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  function copy() {
    if (!intent?.address) return;
    try {
      navigator.clipboard?.writeText(intent.address);
      setCopied(true);
      if (copyTimer.current) clearTimeout(copyTimer.current);
      copyTimer.current = setTimeout(() => setCopied(false), COPIED_FEEDBACK_DURATION);
    } catch (err) {
      console.warn("Clipboard write failed:", err);
    }
  }

  if (pending) {
    return (
      <div className="mt-6 flex justify-center">
        <Loader2 className="animate-spin text-[var(--accent-strong)]" size={20} />
      </div>
    );
  }

  if (!intent) return null;

  const amount = intent.pay_amount || intent.amount || "—";
  const currency = currencyLabel(intent.pay_currency);

  return (
    <div className="mt-6 rounded-2xl border border-[var(--border)] bg-[var(--surface-2)]/40 p-4 space-y-4">
      <div className="flex items-center justify-between">
        <div className="text-xs font-semibold uppercase tracking-wider text-[var(--ink-muted)]">
          {t(lang, "public.cryptoTitle")}
        </div>
        <div className="tabular text-xs font-semibold text-[var(--danger)]">
          {t(lang, "public.cryptoExpires")} {formatCountdown(left)}
        </div>
      </div>

      <div className="flex flex-col items-center gap-3">
        {qrSrc ? (
          <img src={qrSrc} alt={intent.address} className="h-48 w-48 rounded-xl border border-[var(--border)] bg-white p-2" />
        ) : (
          <div className="flex h-48 items-center justify-center text-xs text-[var(--ink-muted)]">QR…</div>
        )}
        {qrSrc ? (
          <button
            type="button"
            onClick={saveQr}
            disabled={saving}
            className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 py-2 text-xs font-semibold text-[var(--ink)] transition-colors hover:bg-[var(--surface-2)] disabled:opacity-60"
          >
            {saving ? <Loader2 size={13} className="animate-spin" /> : <Download size={13} />}
            {t(lang, "public.downloadQr")}
          </button>
        ) : null}
      </div>

      <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-3 space-y-1.5">
        <div className="text-xs text-[var(--ink-muted)]">{t(lang, "public.cryptoAddress")}</div>
        <button
          type="button"
          onClick={copy}
          className="flex w-full items-center justify-between gap-3 rounded-lg border border-[var(--border)] px-3 py-2 text-left text-sm font-mono text-[var(--ink)] hover:bg-[var(--surface-2)]"
        >
          <span className="truncate">{intent.address}</span>
          {copied ? <Check size={15} className="shrink-0 text-[var(--success)]" /> : <Copy size={15} className="shrink-0 text-[var(--ink-muted)]" />}
        </button>
      </div>

      <div className="flex items-center justify-between rounded-xl border border-[var(--border)] bg-[var(--surface)] px-3 py-2.5">
        <span className="text-xs text-[var(--ink-muted)]">{t(lang, "public.cryptoAmount")}</span>
        <span className="tabular text-sm font-semibold text-[var(--ink)]">{amount} {currency}</span>
      </div>

      <p className="text-[11px] leading-relaxed text-[var(--ink-muted)]">
        {t(lang, "public.cryptoHint")}
      </p>
    </div>
  );
}

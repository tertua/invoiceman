import { useEffect, useRef, useState } from "react";
import { Loader2, Copy, Check, Download } from "lucide-react";
import { publicPayApi } from "@/api/publicPay";
import { downloadImage } from "@/lib/download";
import { t } from "@/lib/i18n";

const BASE = 0.91; // 1090s ~ NOWPayments default payment timeout.
const DEFAULT_PAY_CURRENCY = "usdttrc20";

function formatCountdown(left) {
  const m = String(Math.floor(left / 60)).padStart(2, "0");
  const s = String(left % 60).padStart(2, "0");
  return `${m}:${s}`;
}

function payErrorMessage(e, lang) {
  const err = e?.response?.data?.error;
  if (err?.details?.code === "amount_below_minimum") return t(lang, "public.payAmountMinimum");
  if (err?.details?.code === "rate_limited" || e?.response?.status === 429) return t(lang, "public.payRateLimited");
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
// Seam: props {token, lang, onError} + publicPayApi.createTransaction(token,
// method, extra) — both stable, so the widget can be rewritten in place.
// End dev path
export default function CryptoWidget({ token, lang, onError }) {
  const [intent, setIntent] = useState(null);
  const [pending, setPending] = useState(true);
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef(null);
  const [left, setLeft] = useState(Math.round(BASE * 1000));
  const [qrSrc, setQrSrc] = useState("");
  const [saving, setSaving] = useState(false);

  async function saveQr() {
    if (!qrSrc || saving) return;
    setSaving(true);
    const ok = await downloadImage(qrSrc, "usdt-deposit-qr.png");
    setSaving(false);
    if (!ok) window.open(qrSrc, "_blank", "noopener");
  }

  useEffect(() => {
    let cancelled = false;
    setPending(true);
    publicPayApi
      .createTransaction(token, "crypto", { pay_currency: DEFAULT_PAY_CURRENCY })
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
  }, [token, lang, onError, copyTimer]);

  useEffect(() => {
    let mounted = true;
    if (intent?.address) {
      import("qrcode").then(({ default: QRCode }) => {
        if (!mounted) return;
        QRCode.toDataURL(intent.address, { margin: 1, width: 200 }).then(setQrSrc).catch(() => {});
      });
    }
    return () => {
      mounted = false;
    };
  }, [intent]);

  useEffect(() => {
    const start = Date.now();
    const total = Math.round(BASE * 1000);
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
      copyTimer.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
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
          {lang === "id" ? "Bayar dengan USDT (TRC20)" : "Pay with USDT (TRC20)"}
        </div>
        <div className="tabular text-xs font-semibold text-[var(--danger)]">
          {lang === "id" ? "Sisa waktu" : "Expires in"} {formatCountdown(left)}
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
        <div className="text-xs text-[var(--ink-muted)]">{lang === "id" ? "Alamat deposit" : "Deposit address"}</div>
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
        <span className="text-xs text-[var(--ink-muted)]">{lang === "id" ? "Jumlah" : "Amount"}</span>
        <span className="tabular text-sm font-semibold text-[var(--ink)]">{amount} {currency}</span>
      </div>

      <p className="text-[11px] leading-relaxed text-[var(--ink-muted)]">
        {lang === "id"
          ? "Kirim jumlah persis di atas ke alamat deposit. Pembayaran terverifikasi otomatis setelah konfirmasi jaringan."
          : "Send the exact amount above to the deposit address. Payment is verified automatically once the network confirms it."}
      </p>
    </div>
  );
}

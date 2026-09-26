import { useEffect, useRef, useState } from "react";
import { Download, Loader2 } from "lucide-react";
import { createPublicTransaction, newIntentKey, qrDownloadUrl } from "@/api/publicIntent";
import { loadMidtransSnap } from "@/lib/midtrans";
import { downloadImage } from "@/lib/download";
import { t } from "@/lib/i18n";
import { formatMoney } from "@/lib/utils";

const DEFAULT_TTL = 900;
const REVOKE_DELAY = 4000; // ms to wait before revoking blob URL

function countdownFrom(expiresAt) {
  if (!expiresAt) return DEFAULT_TTL;
  const at = Date.parse(expiresAt);
  if (Number.isNaN(at)) return DEFAULT_TTL;
  return Math.max(0, Math.round((at - Date.now()) / 1000));
}

function formatCountdown(left) {
  const m = String(Math.floor(left / 60)).padStart(2, "0");
  const s = String(left % 60).padStart(2, "0");
  return `${m}:${s}`;
}

// Dev path — on-page QRIS widget (scaffold, Gopay acquirer on purpose)
//   [done]  QRIS charge via the Core API (acquirer=gopay), QR image + expiry
//           countdown, Snap checkout fallback when the Core API is unavailable;
//           an expired QR is hidden and re-requested via "new QR" (backend
//           recharges under a -rN order id instead of resending a dead QR)
//   [next]  acquirer stays a backend decision; the widget only renders qr_image
//   [later] poll the intent status inline instead of relying on the page poll
// Seam: props {token, lang, amount, currency, gateway, onError, onPaid} +
// createPublicTransaction(token, "qris", undefined, idempotencyKey) —
// one UUID per nonce, stable across StrictMode double-effects so the retry
// replays instead of double-charging; a 409 (sibling request in flight)
// retries once with the same key. Stable, so the widget can be
// rewritten in place.
// End dev path
export default function QrisWidget({ token, lang, amount, currency, gateway, onError, onPaid }) {
  const [intent, setIntent] = useState(null);
  const [pending, setPending] = useState(true);
  const [left, setLeft] = useState(DEFAULT_TTL);
  const [nonce, setNonce] = useState(0);
  const [saving, setSaving] = useState(false);
  const keysRef = useRef({});
  const clientKey = gateway?.client_key;
  const isProd = gateway?.is_production;

  function qrFileName(disposition) {
    const m = /filename="([A-Za-z0-9-_]+\.png)"/.exec(disposition || "");
    if (m) return m[1];
    const id = String(intent?.order_id || "").replace(/[^A-Za-z0-9-_]/g, "").slice(0, 48);
    return `${id || "qris"}.png`;
  }

  async function saveQr() {
    if (!intent?.payment_url || saving) return;
    setSaving(true);
    try {
      const res = await fetch(qrDownloadUrl(token));
      const ct = res.headers.get("content-type") || "";
      if (res.ok && ct.startsWith("image/")) {
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = qrFileName(res.headers.get("content-disposition"));
        document.body.appendChild(a);
        a.click();
        a.remove();
        setTimeout(() => URL.revokeObjectURL(url), REVOKE_DELAY);
        setSaving(false);
        return;
      }
    } catch (err) {
      console.warn("QR download via backend failed, trying direct image download:", err);
    }
    const ok = await downloadImage(intent.payment_url, qrFileName(""));
    setSaving(false);
    if (!ok) window.open(intent.payment_url, "_blank", "noopener");
  }

  // One idempotency key per QR generation. The ref survives StrictMode's
  // setup→cleanup→setup cycle, so the double-fired effect sends the same
  // key twice and the backend replays the first charge instead of opening
  // a second one at the gateway. A new nonce ("new QR") mints a new key.
  function intentKey(n) {
    if (!keysRef.current[n]) keysRef.current[n] = newIntentKey(n);
    return keysRef.current[n];
  }

  useEffect(() => {
    let cancelled = false;
    setPending(true);
    const key = intentKey(nonce);
    let retried = false;
    async function attempt() {
      let res;
      try {
        res = await createPublicTransaction(token, "qris", undefined, key);
      } catch (e) {
        const status = e?.status ?? e?.original?.response?.status;
        if (!cancelled && status === 409 && !retried) {
          retried = true;
          await new Promise((r) => setTimeout(r, 1200));
          if (!cancelled) return attempt();
        }
        if (!cancelled) onError(t(lang, "public.qrisCreateFailed"));
        if (!cancelled) setPending(false);
        return;
      }
      if (cancelled) return;
      if (res.payment_url) {
        setIntent(res);
        setPending(false);
        return;
      }
      if (res.snap_token && clientKey) {
        setPending(false);
        const snap = await loadMidtransSnap(isProd);
        snap.pay(res.snap_token, {
          onClose: () => {},
          onError: () => {},
          onSuccess: () => onPaid?.(),
        });
        return;
      }
      if (res.redirect_url) {
        window.location.href = res.redirect_url;
        return;
      }
      onError(t(lang, "public.qrisCreateFailed"));
      setPending(false);
    }
    attempt();
    return () => {
      cancelled = true;
    };
  }, [token, lang, clientKey, isProd, onError, onPaid, nonce]);

  useEffect(() => {
    if (!intent) return undefined;
    const total = countdownFrom(intent.expires_at);
    setLeft(total);
    const start = Date.now();
    const timer = setInterval(() => {
      const remaining = Math.max(0, Math.round((start + total * 1000 - Date.now()) / 1000));
      setLeft(remaining);
      if (remaining <= 0) clearInterval(timer);
    }, 1000);
    return () => clearInterval(timer);
  }, [intent]);

  if (pending) {
    return (
      <div className="mt-6 flex justify-center">
        <Loader2 className="animate-spin text-[var(--accent-strong)]" size={20} />
      </div>
    );
  }

  if (!intent) return null;

  const expired = left <= 0;

  return (
    <div className="mt-6 rounded-2xl border border-[var(--border)] bg-[var(--surface-2)]/40 p-4 space-y-4">
      <div className="flex items-center justify-between">
        <div className="text-xs font-semibold uppercase tracking-wider text-[var(--ink-muted)]">
          {t(lang, "public.qrisTitle")}
        </div>
        <div className="tabular text-xs font-semibold text-[var(--danger)]">
          {expired ? t(lang, "public.qrisExpired") : `${t(lang, "public.qrisExpires")} ${formatCountdown(left)}`}
        </div>
      </div>

      {expired ? (
        <div className="flex flex-col items-center gap-3 py-4">
          <p className="text-sm text-[var(--ink-muted)]">{t(lang, "public.qrisExpiredHint")}</p>
          <button
            type="button"
            disabled={pending}
            onClick={() => {
              setPending(true);
              setNonce((n) => n + 1);
            }}
            className="rounded-2xl border border-[var(--accent)]/50 bg-[var(--surface)] px-5 py-2.5 text-sm font-semibold text-[var(--ink)] transition-colors hover:bg-[var(--surface-2)] disabled:opacity-60"
          >
            {t(lang, "public.qrisRegenerate")}
          </button>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-3">
          <img
            src={intent.payment_url}
            alt={t(lang, "public.qrisQrAlt")}
            className="h-56 w-56 rounded-xl border border-[var(--border)] bg-white p-2"
          />
          <button
            type="button"
            onClick={saveQr}
            disabled={saving}
            className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 py-2 text-xs font-semibold text-[var(--ink)] transition-colors hover:bg-[var(--surface-2)] disabled:opacity-60"
          >
            {saving ? <Loader2 size={13} className="animate-spin" /> : <Download size={13} />}
            {t(lang, "public.downloadQr")}
          </button>
        </div>
      )}

      {amount ? (
        <div className="flex items-center justify-between rounded-xl border border-[var(--border)] bg-[var(--surface)] px-3 py-2.5">
          <span className="text-xs text-[var(--ink-muted)]">{t(lang, "public.qrisAmount")}</span>
          <span className="tabular text-sm font-semibold text-[var(--ink)]">{formatMoney(amount, currency)}</span>
        </div>
      ) : null}

      {expired ? null : (
        <p className="text-[11px] leading-relaxed text-[var(--ink-muted)]">{t(lang, "public.qrisHint")}</p>
      )}
    </div>
  );
}
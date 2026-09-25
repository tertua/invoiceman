import { Loader2, ArrowRight, ShieldCheck } from "lucide-react";
import { t } from "@/lib/i18n";

// MethodPicker lets the payer choose how to pay; the gateway intent (and its
// paylink) is only created after a method is chosen. The invoice total is
// already shown above, so each button carries only the method name — the
// provider's charged amount/currency is not repeated per button. Crypto hands off
// to the on-page CryptoWidget instead of redirecting.
//
// Dev path — method picker (scaffold, flat by design today)
//   [done]  one row per provider method, name only
//   [next]  group rows by charge currency (IDR domestic / USD international)
//           and render m.currency + m.amount — both already in the payload
//   [later] currency toggle on the pay page
// Seam: props {methods, lang, onPick, pending, error} — the file can be
// replaced wholesale without touching PublicPay.jsx.
// End dev path
export default function MethodPicker({ methods, lang, onPick, pending, error }) {
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
              {pending === m.id ? (
                <Loader2 size={15} className="animate-spin text-[var(--accent-strong)]" />
              ) : (
                <ArrowRight size={15} className="text-[var(--accent-strong)]" />
              )}
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

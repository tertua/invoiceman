import { CRYPTO_ASSETS } from "@/lib/cryptoAssets";
import { t } from "@/lib/i18n";

export default function CryptoAssetPicker({ value, onChange, lang, disabled }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs text-[var(--ink-muted)]">{t(lang, "public.cryptoAsset")}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className="w-full rounded-xl border border-[var(--border)] bg-[var(--surface)] px-3 py-2.5 text-sm text-[var(--ink)] disabled:opacity-60"
      >
        {CRYPTO_ASSETS.map((a) => (
          <option key={a.code} value={a.code}>
            {a.network ? `${a.ticker} · ${a.network}` : a.ticker}
          </option>
        ))}
      </select>
    </label>
  );
}

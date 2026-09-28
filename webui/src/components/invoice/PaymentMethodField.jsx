import { useLang } from "@/context/LangContext";
import { PAYMENT_METHODS } from "@/lib/paymentMethods";

const selectClass =
  "h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15 disabled:opacity-50 disabled:cursor-not-allowed";

export function PaymentMethodField({ value, disabled, onChange }) {
  const { t } = useLang();
  return (
    <>
      <select
        className={selectClass}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
      >
        <option value="">{t("invEditor.noMethod")}</option>
        {PAYMENT_METHODS.map((m) => (
          <option key={m} value={m}>{m}</option>
        ))}
      </select>
    </>
  );
}

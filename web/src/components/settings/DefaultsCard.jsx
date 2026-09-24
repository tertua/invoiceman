import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { useLang } from "@/context/LangContext";
import { CURRENCIES } from "@/lib/utils";

function FieldLabel({ children }) {
  return <label className="text-xs font-medium text-[var(--ink-muted)] mb-1.5 block">{children}</label>;
}

// Invoicing defaults card, including the manual USD→IDR rate used when an
// online gateway charges a different currency than the invoice.
export default function DefaultsCard({ form, set, selectClass }) {
  const { t } = useLang();
  return (
    <Card padding="lg">
      <CardHeader>
        <div>
          <CardTitle className="text-base">{t("settings.defaults")}</CardTitle>
          <CardDescription className="mt-1">{t("settings.defaultsDesc")}</CardDescription>
        </div>
      </CardHeader>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div>
          <FieldLabel>{t("settings.defaultCurrency")}</FieldLabel>
          <select className={selectClass} value={form.currency} onChange={set("currency")}>
            {CURRENCIES.map((c) => (
              <option key={c.code} value={c.code}>
                {c.code}
              </option>
            ))}
          </select>
        </div>
        <div>
          <FieldLabel>{t("settings.defaultTax")}</FieldLabel>
          <Input type="number" min="0" step="0.1" value={form.tax_rate} onChange={set("tax_rate")} className="tabular" />
        </div>
        <div>
          <FieldLabel>{t("settings.prefix")}</FieldLabel>
          <Input value={form.invoice_prefix} onChange={set("invoice_prefix")} placeholder="INV-" />
        </div>
        <div className="sm:col-span-3">
          <FieldLabel>{t("settings.usdToIdr")}</FieldLabel>
          <Input
            type="number"
            min="0"
            step="0.01"
            value={form.usd_to_idr}
            onChange={set("usd_to_idr")}
            className="tabular"
            placeholder="18000"
          />
          <p className="text-[11px] text-[var(--ink-muted)] mt-1.5">{t("settings.usdToIdrHint")}</p>
        </div>
      </div>
    </Card>
  );
}

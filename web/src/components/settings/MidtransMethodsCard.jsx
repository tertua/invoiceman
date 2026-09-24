import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { useLang } from "@/context/LangContext";
import { useMidtransMethods } from "@/hooks/useMidtransMethods";

// firstMethod picks the stored id when known, defaulting to gopay for empty
// or legacy multi-method values. The public pay page then offers exactly one
// method, so one payer click can only open one gateway intent.
function firstMethod(csv, methods) {
  const ids = String(csv || "")
    .split(",")
    .map((m) => m.trim())
    .filter(Boolean);
  const known = new Set((methods || []).map((m) => m.id));
  return ids.find((id) => known.has(id)) || "gopay";
}

// MidtransMethodsCard lets an owner pick the single method Midtrans offers
// (and the public pay page shows). One method means one intent: payers can no
// longer open a separate pending Midtrans transaction per method.
export default function MidtransMethodsCard({ value, onChange }) {
  const { t } = useLang();
  const { data: methods, isLoading } = useMidtransMethods();
  const all = methods || [];
  const selected = firstMethod(value, all);

  return (
    <Card padding="lg">
      <CardHeader>
        <div>
          <CardTitle className="text-base">{t("settings.midtransMethods")}</CardTitle>
          <CardDescription className="mt-1">{t("settings.midtransMethodsDesc")}</CardDescription>
        </div>
      </CardHeader>
      {isLoading ? (
        <Skeleton className="h-10 w-full" />
      ) : (
        <select
          className="h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15"
          value={selected}
          onChange={(e) => onChange(e.target.value)}
        >
          {all.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      )}
      <p className="text-[11px] text-[var(--ink-muted)] mt-3">{t("settings.midtransMethodsHint")}</p>
    </Card>
  );
}

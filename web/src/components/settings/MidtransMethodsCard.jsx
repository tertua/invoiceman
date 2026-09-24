import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Checkbox } from "@/components/ui/Checkbox";
import { Skeleton } from "@/components/ui/Skeleton";
import { useLang } from "@/context/LangContext";
import { useMidtransMethods } from "@/hooks/useMidtransMethods";

// parseMethods turns the stored CSV into a set. Empty means "all enabled".
function parseMethods(csv) {
  return new Set(
    String(csv || "")
      .split(",")
      .map((m) => m.trim())
      .filter(Boolean)
  );
}

// MidtransMethodsCard lets an owner limit which methods Midtrans offers (and
// the public pay page shows). An empty selection means "all Midtrans
// methods", which is the default, so accounts that never touch this keep
// every method. The last checked method can't be removed: an empty list
// would silently mean "all".
export default function MidtransMethodsCard({ value, onChange }) {
  const { t } = useLang();
  const { data: methods, isLoading } = useMidtransMethods();
  const all = methods || [];
  const selected = parseMethods(value);
  const allOn = selected.size === 0;
  const isOn = (id) => allOn || selected.has(id);

  function toggle(id) {
    const next = new Set(allOn ? all.map((m) => m.id) : selected);
    if (next.has(id)) {
      if (next.size === 1) return;
      next.delete(id);
    } else {
      next.add(id);
    }
    const coversAll = all.length > 0 && all.every((m) => next.has(m.id));
    onChange(coversAll ? "" : all.filter((m) => next.has(m.id)).map((m) => m.id).join(","));
  }

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
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2.5">
          {all.map((m) => (
            <Checkbox key={m.id} checked={isOn(m.id)} onChange={() => toggle(m.id)} label={m.name} />
          ))}
        </div>
      )}
      <p className="text-[11px] text-[var(--ink-muted)] mt-3">{t("settings.midtransMethodsHint")}</p>
    </Card>
  );
}

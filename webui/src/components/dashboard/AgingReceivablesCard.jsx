import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { useLang } from "@/context/LangContext";
import { formatMoney } from "@/lib/utils";
import { agingBucketKey } from "@/lib/chartLabels";

// Compact receivables-aging summary: five buckets (not due, 1-30, 31-60,
// 61-90, 90+) as CSS bars, no recharts. The full aging chart still lives in
// DashboardCharts (lazy chunk); this card is the at-a-glance version that must
// stay in the entry chunk, so it draws bars from tokens alone.
const ORDER = ["current", "d1_30", "d31_60", "d61_90", "d90_plus"];
const LABEL_KEY = {
  current: "dash.agingCurrent",
  d1_30: "dash.aging30",
  d31_60: "dash.aging60",
  d61_90: "dash.aging90",
  d90_plus: "dash.aging90plus",
};
const BAR_COLOR = {
  current: "var(--accent)",
  d1_30: "var(--warning)",
  d31_60: "var(--warning)",
  d61_90: "var(--danger)",
  d90_plus: "var(--danger)",
};

export function AgingReceivablesCard({ aging, className }) {
  const { t } = useLang();
  const byKey = new Map((Array.isArray(aging) ? aging : []).map((point) => [agingBucketKey(point), Number(point?.value) || 0]));
  const rows = ORDER.map((key) => ({ key, value: byKey.get(key) || 0 }));
  const total = rows.reduce((sum, row) => sum + row.value, 0);
  const hasData = total > 0;

  return (
    <Card padding="lg" className={className}>
      <CardHeader>
        <div>
          <CardTitle>{t("dash.receivablesAging")}</CardTitle>
          <CardDescription>{t("dash.receivablesAgingDesc")}</CardDescription>
        </div>
      </CardHeader>
      {!aging ? (
        <div className="space-y-3 pt-1">
          {ORDER.map((key) => (
            <Skeleton key={key} className="h-8 rounded-xl" />
          ))}
        </div>
      ) : !hasData ? (
        <div className="py-6 text-center text-sm text-[var(--ink-muted)]">{t("dash.allCaughtUp")}</div>
      ) : (
        <div className="space-y-3 pt-1">
          {rows.map((row) => (
            <div key={row.key}>
              <div className="flex items-center justify-between text-sm mb-1.5">
                <span className="text-[var(--ink-muted)]">{t(LABEL_KEY[row.key])}</span>
                <span className="tabular font-semibold text-[var(--ink)]">{formatMoney(row.value)}</span>
              </div>
              <div className="h-2 rounded-full bg-[var(--surface-2)] overflow-hidden">
                <div
                  className="h-full rounded-full"
                  style={{ width: `${Math.max(row.value > 0 ? 4 : 0, (row.value / total) * 100)}%`, background: BAR_COLOR[row.key] }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

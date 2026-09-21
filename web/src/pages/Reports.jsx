import { lazy, Suspense } from "react";
import {
  Wallet,
  Receipt,
  TrendingUp,
  Clock,
  Download,
  BarChart3,
} from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/Button";
import { StatCard } from "@/components/dashboard/StatCard";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import { useReports } from "@/hooks/useFeatures";
import { todayDateInput } from "@/lib/utils";
import { useLang } from "@/context/LangContext";
import { formatMoney } from "@/lib/utils";

const ReportsCharts = lazy(() => import("@/components/reports/ReportsCharts"));

export default function Reports() {
  const { t } = useLang();
  const { data, isLoading } = useReports();

  if (isLoading) return <ReportsSkeleton />;
  if (!data) return <EmptyState icon={BarChart3} title={t("reports.noData")} description={t("reports.noDataDesc")} />;

  const {
    totals = {},
    monthly = [],
    aging = [],
    topClients = [],
    statusBreakdown = [],
  } = data;
  const statusData = statusBreakdown.filter((s) => s.value > 0);
  const agingHasData = aging.some((a) => a.value > 0);
  const maxClient = Math.max(1, ...topClients.map((c) => c.billed));

  function exportCSV() {
    const rows = [
      [t("reports.csv.metric"), t("reports.csv.value")],
      [t("reports.csv.totalRevenue"), totals.revenue],
      [t("reports.csv.totalExpenses"), totals.expenses],
      [t("reports.csv.netProfit"), totals.netProfit],
      [t("reports.csv.outstanding"), totals.outstanding],
      [],
      [t("reports.csv.month"), t("reports.csv.revenue"), t("reports.csv.expenses")],
      ...monthly.map((m) => [m.label, m.revenue, m.expenses]),
      [],
      [t("reports.csv.agingBucket"), t("reports.csv.amount")],
      ...aging.map((a) => [a.bucket, a.value]),
      [],
      [t("reports.csv.topClient"), t("reports.csv.billed"), t("reports.csv.paid")],
      ...topClients.map((c) => [c.name, c.billed, c.paid]),
    ];
    const csv = rows.map((r) => r.map((v) => `"${String(v ?? "").replace(/"/g, '""')}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `invoicer-report-${todayDateInput()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t("reports.title")}
        description={t("reports.desc")}
        actions={
          <Button variant="outline" onClick={exportCSV}>
            <Download size={15} /> {t("reports.export")}
          </Button>
        }
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <StatCard label={t("reports.totalRevenue")} value={formatMoney(totals.revenue)} icon={Wallet} accent />
        <StatCard label={t("reports.totalExpenses")} value={formatMoney(totals.expenses)} icon={Receipt} />
        <StatCard label={t("reports.netProfit")} value={formatMoney(totals.netProfit)} icon={TrendingUp} />
        <StatCard label={t("reports.outstanding")} value={formatMoney(totals.outstanding)} icon={Clock} />
      </div>

      <Suspense fallback={<ReportsChartsFallback />}>
        <ReportsCharts monthly={monthly} statusData={statusData} aging={aging} agingHasData={agingHasData} topClients={topClients} maxClient={maxClient} />
      </Suspense>
    </div>
  );
}

function ReportsSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-10 w-56 rounded-2xl" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-[120px] rounded-2xl" />)}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <Skeleton className="lg:col-span-2 h-[360px] rounded-3xl" />
        <Skeleton className="h-[360px] rounded-3xl" />
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        <Skeleton className="h-[320px] rounded-3xl" />
        <Skeleton className="h-[320px] rounded-3xl" />
      </div>
    </div>
  );
}

function ReportsChartsFallback() {
  return (
    <>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <Skeleton className="lg:col-span-2 h-[360px] rounded-3xl" />
        <Skeleton className="h-[360px] rounded-3xl" />
      </div>
      <Skeleton className="h-[320px] rounded-3xl" />
    </>
  );
}

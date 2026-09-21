import { ResponsiveContainer, AreaChart, Area, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Cell, PieChart, Pie } from "recharts";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { useLang } from "@/context/LangContext";
import { formatMoney, localizeAgingBuckets, localizeMonthLabels } from "@/lib/utils";

const T1 = "#2dd4bf";
const T2 = "#14b8a6";
const T3 = "#0f766e";
const STATUS_COLORS = { draft: "#94a3b8", sent: "#5eead4", overdue: "var(--danger)", paid: T2 };
const tooltipStyle = { background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 14, fontSize: 12, color: "var(--ink)", boxShadow: "var(--shadow-hover)" };

export function DashboardCharts({ series, reports, total, collections }) {
  return <><div className="grid grid-cols-1 lg:grid-cols-12 gap-5"><div className="lg:col-span-8"><RevenueChart series={series} /></div><div className="lg:col-span-4">{collections}</div></div><div className="grid grid-cols-1 lg:grid-cols-12 gap-5"><div className="lg:col-span-4"><StatusDonutCard reports={reports} total={total} /></div><div className="lg:col-span-4"><AgingCard reports={reports} /></div></div></>;
}

export function DashboardChartsFallback() {
  return <><Skeleton className="h-[360px] rounded-3xl" /><div className="grid grid-cols-1 lg:grid-cols-2 gap-5"><Skeleton className="h-[240px] rounded-3xl" /><Skeleton className="h-[240px] rounded-3xl" /></div></>;
}

function RevenueChart({ series }) {
  const { t } = useLang();
  const data = localizeMonthLabels(series);
  const hasRevenue = data.some((item) => item.revenue > 0);
  return <Card padding="lg" className="h-full"><CardHeader><div><CardTitle>{t("dash.revenue")}</CardTitle><CardDescription>{t("dash.revenueDesc")}</CardDescription></div></CardHeader>{hasRevenue ? <ResponsiveContainer width="100%" height={280}><AreaChart data={data} margin={{ top: 10, right: 8, bottom: 0, left: -10 }}><defs><linearGradient id="revArea" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor={T1} stopOpacity={0.35} /><stop offset="100%" stopColor={T2} stopOpacity={0} /></linearGradient><linearGradient id="revStroke" x1="0" y1="0" x2="1" y2="0"><stop offset="0%" stopColor={T1} /><stop offset="100%" stopColor={T3} /></linearGradient></defs><CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} /><XAxis dataKey="label" axisLine={false} tickLine={false} tick={{ fill: "var(--ink-muted)", fontSize: 12 }} /><YAxis axisLine={false} tickLine={false} tick={{ fill: "var(--ink-muted)", fontSize: 12 }} tickFormatter={(v) => (v >= 1000 ? `${v / 1000}k` : v)} /><Tooltip contentStyle={tooltipStyle} formatter={(v) => [formatMoney(v), t("dash.revenue")]} /><Area type="monotone" dataKey="revenue" stroke="url(#revStroke)" strokeWidth={3} fill="url(#revArea)" dot={{ r: 3, fill: T2, strokeWidth: 0 }} activeDot={{ r: 5, fill: T2 }} /></AreaChart></ResponsiveContainer> : <div className="h-[280px] flex flex-col items-center justify-center text-center"><div className="font-display text-sm font-semibold mb-1">{t("dash.noPaid")}</div><div className="text-xs text-[var(--ink-muted)]">{t("dash.markPaid")}</div></div>}</Card>;
}

function StatusDonutCard({ reports, total }) {
  const { t } = useLang();
  const data = (reports?.statusBreakdown || []).filter((item) => item.value > 0);
  return <Card padding="lg" className="h-full flex flex-col"><CardHeader><div><CardTitle>{t("dash.invoiceStatus")}</CardTitle><CardDescription>{t("dash.byAmount")}</CardDescription></div></CardHeader>{!reports ? <ChartSkeleton /> : data.length ? <div className="flex-1 flex items-center gap-5"><div className="relative h-[172px] w-[172px] shrink-0"><ResponsiveContainer width="100%" height="100%"><PieChart><Pie data={data} dataKey="value" innerRadius={54} outerRadius={80} paddingAngle={2} stroke="none">{data.map((item) => <Cell key={item.key} fill={STATUS_COLORS[item.key]} />)}</Pie><Tooltip contentStyle={tooltipStyle} formatter={(v, n) => [formatMoney(v), t("status." + (n || "draft").toLowerCase())]} /></PieChart></ResponsiveContainer><div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none"><span className="font-display text-2xl font-semibold text-[var(--ink)]">{total}</span><span className="text-[10px] text-[var(--ink-muted)]">{t("dash.invoices")}</span></div></div><div className="flex-1 min-w-0 space-y-3">{data.map((item) => <div key={item.key} className="flex items-center gap-2 text-sm"><span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ background: STATUS_COLORS[item.key] }} /><span className="text-[var(--ink-muted)] flex-1">{t("status." + (item.key || "draft"))}</span><span className="tabular font-semibold text-[var(--ink)]">{formatMoney(item.value)}</span></div>)}</div></div> : <div className="flex-1 flex items-center justify-center text-sm text-[var(--ink-muted)]">{t("dash.noInvoices")}</div>}</Card>;
}

function AgingCard({ reports }) {
  const { t } = useLang();
  const aging = localizeAgingBuckets(reports?.aging, t);
  const hasData = aging.some((item) => item.value > 0);
  return <Card padding="lg" className="h-full flex flex-col"><CardHeader><div><CardTitle>{t("dash.aging")}</CardTitle><CardDescription>{t("dash.agingDesc")}</CardDescription></div></CardHeader>{!reports ? <ChartSkeleton /> : hasData ? <div className="flex-1 min-h-[180px]"><ResponsiveContainer width="100%" height="100%"><BarChart data={aging} margin={{ top: 8, right: 4, bottom: 0, left: -14 }}><CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} /><XAxis dataKey="bucket" axisLine={false} tickLine={false} tick={{ fill: "var(--ink-muted)", fontSize: 10 }} /><YAxis axisLine={false} tickLine={false} tick={{ fill: "var(--ink-muted)", fontSize: 11 }} tickFormatter={(v) => (v >= 1000 ? `${v / 1000}k` : v)} /><Tooltip cursor={{ fill: "var(--surface-2)" }} contentStyle={tooltipStyle} formatter={(v) => [formatMoney(v), t("dash.amount")]} /><Bar dataKey="value" radius={[6, 6, 0, 0]} maxBarSize={40}>{aging.map((_, index) => <Cell key={index} fill={index === 0 ? T2 : index >= 3 ? "var(--danger)" : "var(--warning)"} />)}</Bar></BarChart></ResponsiveContainer></div> : <div className="flex-1 flex items-center justify-center text-center text-sm text-[var(--ink-muted)]">{t("dash.allCaughtUp")}</div>}</Card>;
}

function ChartSkeleton() { return <Skeleton className="h-[150px] rounded-2xl" />; }

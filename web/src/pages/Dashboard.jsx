import { lazy, Suspense, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Wallet,
  Clock,
  TrendingUp,
  AlertTriangle,
  Plus,
  Sparkles,
  Loader2,
  ArrowRight,
  Users,
} from "lucide-react";
import { StatCard } from "@/components/dashboard/StatCard";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { QueryError } from "@/components/ui/QueryError";
import { Button } from "@/components/ui/Button";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { StatusBadge } from "@/components/ui/Badge";
import { useDashboard } from "@/hooks/useDashboard";
import { useReports } from "@/hooks/useReports";
import { aiApi, isAiUnavailable, isAiFailure, isAiRateLimited } from "@/api/ai";
import { useLang } from "@/context/LangContext";
import { useAuth } from "@/context/AuthContext";
import { useSettings } from "@/hooks/useSettings";
import { formatMoney, formatDate } from "@/lib/utils";
const DashboardCharts = lazy(() => import("@/components/dashboard/DashboardCharts").then((module) => ({ default: module.DashboardCharts })));

// Logo palette
const T1 = "#2dd4bf"; // teal-400
const T2 = "#14b8a6"; // teal-500
const T3 = "#0f766e"; // teal-700


export default function Dashboard() {
  const nav = useNavigate();
  const { t } = useLang();
  const { data: settings } = useSettings();
  const currency = settings?.currency || "IDR";
  const { data, isLoading, error } = useDashboard(currency);
  const { data: reports } = useReports(currency);

  if (isLoading) return <DashboardSkeleton />;
  if (error) return <QueryError error={error} />;

  const { stats, revenueSeries, recentInvoices } = data || {};

  if (!stats?.invoiceCount) {
    return (
      <EmptyState
        icon={Plus}
        title={t("dash.welcomeTitle")}
        description={t("dash.welcomeDesc")}
        action={
          <Button variant="accent" size="lg" onClick={() => nav("/invoices/new")}>
            <Plus size={16} /> {t("dash.createFirst")}
          </Button>
        }
      />
    );
  }

  const collected = Number(stats.totalRevenue) || 0;
  const owed = Number(stats.outstanding) || 0;
  const collectionRate = collected + owed > 0 ? Math.round((collected / (collected + owed)) * 100) : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h2 className="font-display text-2xl font-semibold tracking-tight">{t("dash.overview")}</h2>
          <p className="text-sm text-[var(--ink-muted)] mt-1">
            {t("dash.counts", { invoices: stats.invoiceCount, clients: stats.clientCount })}
          </p>
        </div>
        <Button variant="accent" onClick={() => nav("/invoices/new")}>
          <Plus size={16} /> {t("dash.createInvoice")}
        </Button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <StatCard label={t("dash.totalRevenue")} value={formatMoney(stats.totalRevenue)} icon={Wallet} accent />
        <StatCard label={t("dash.outstanding")} value={formatMoney(stats.outstanding)} icon={Clock} />
        <StatCard label={t("dash.paidThisMonth")} value={formatMoney(stats.paidThisMonth)} icon={TrendingUp} />
        <StatCard label={t("dash.overdue")} value={stats.overdueCount} suffix={stats.overdueTotal ? formatMoney(stats.overdueTotal) : null} icon={AlertTriangle} />
      </div>

      <AISummaryCard stats={stats} />

      <Suspense fallback={<ChartFallback />}>
        <DashboardCharts
          series={revenueSeries}
          reports={reports}
          total={stats.invoiceCount}
          collections={<CollectionsCard rate={collectionRate} collected={stats.totalRevenue} outstanding={stats.outstanding} />}
        />
      </Suspense>
      <TopClientsCard reports={reports} />

      <RecentInvoices invoices={recentInvoices} onOpen={(id) => nav(`/invoices/${id}`)} />
    </div>
  );
}

/* ─────────────────── AI summary ─────────────────── */
// The last generated summary survives menu switches and reloads via
// localStorage, scoped per user and language; regenerating overwrites it.
const SUMMARY_KEY_PREFIX = "invoiceman:ai-summary:";

function summaryKey(userId, lang) {
  return `${SUMMARY_KEY_PREFIX}${userId || "anon"}:${lang === "id" ? "id" : "en"}`;
}

function loadCachedSummary(key) {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return "";
    const parsed = JSON.parse(raw);
    return typeof parsed?.summary === "string" ? parsed.summary : "";
  } catch {
    return "";
  }
}

function AISummaryCard({ stats }) {
  const { t, lang } = useLang();
  const { user } = useAuth();
  const key = summaryKey(user?.id, lang);
  const [summary, setSummary] = useState(() => loadCachedSummary(key));
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");
  const [unavailable, setUnavailable] = useState(false);

  // Pick up the cache for the active user/language (e.g. after switching
  // language or account without a full reload).
  useEffect(() => {
    setSummary(loadCachedSummary(key));
  }, [key]);

  async function generate() {
    setLoading(true);
    setErr("");
    setUnavailable(false);
    try {
      const res = await aiApi.businessSummary();
      setSummary(res.summary);
      try {
        localStorage.setItem(key, JSON.stringify({ summary: res.summary, at: Date.now() }));
      } catch {
        /* private mode / quota — in-memory summary still shows */
      }
    } catch (e) {
      setUnavailable(isAiUnavailable(e));
      if (e.status !== 401) setErr(isAiUnavailable(e) ? t("ai.unavailable") : isAiRateLimited(e) ? t("ai.rateLimited") : isAiFailure(e) ? t("ai.failed") : e.message || t("dash.generateFailed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card padding="lg" className="relative overflow-hidden">
      <div className="relative flex items-start gap-4">
        <div className="h-11 w-11 rounded-2xl flex items-center justify-center text-white shrink-0 shadow-[0_8px_20px_-6px_rgba(13,148,136,0.6)]" style={{ background: `linear-gradient(135deg,${T1},${T3})` }}>
          <Sparkles size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="font-display text-sm font-semibold tracking-tight">{t("dash.aiSummary")}</div>
              <div className="text-xs text-[var(--ink-muted)]">{t("dash.aiSummaryDesc")}</div>
            </div>
            <Button variant="soft" size="sm" onClick={generate} disabled={loading || unavailable}>
              {loading ? <Loader2 size={13} className="animate-spin" /> : <Sparkles size={13} />}
              {summary ? t("dash.regenerate") : t("dash.generate")}
            </Button>
          </div>
          {err && <p className={`text-sm mt-3 ${unavailable ? "text-[var(--ink-muted)]" : "text-[var(--danger)]"}`}>{err}</p>}
          {summary ? (
            <p className="text-[15px] leading-relaxed text-[var(--ink)] mt-3">{summary}</p>
          ) : (
            !err && (
              <p className="text-sm text-[var(--ink-muted)] mt-3">
                {stats.overdueCount
                  ? (stats.overdueCount > 1
                      ? t("dash.overdueLinePlural", { n: stats.overdueCount, amount: formatMoney(stats.overdueTotal) })
                      : t("dash.overdueLine", { n: stats.overdueCount, amount: formatMoney(stats.overdueTotal) }))
                  : ""}
                {t("dash.generateSummary")}
              </p>
            )
          )}
        </div>
      </div>
    </Card>
  );
}

/* ─────────────────── Collections half-donut gauge ─────────────────── */
// Custom SVG semicircle gauge — full control over radius/thickness so the
// number always sits cleanly in the hollow (unlike Recharts' capped radius).
function HalfGauge({ value }) {
  const r = 42;
  const cx = 50;
  const cy = 50;
  const len = Math.PI * r; // length of a semicircle
  const p = Math.max(0, Math.min(100, Number(value) || 0)) / 100;
  const d = `M ${cx - r} ${cy} A ${r} ${r} 0 0 1 ${cx + r} ${cy}`;
  return (
    <svg viewBox="0 0 100 56" className="w-full block">
      <defs>
        <linearGradient id="gaugeGrad" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor={T1} />
          <stop offset="100%" stopColor={T3} />
        </linearGradient>
      </defs>
      <path d={d} fill="none" stroke="var(--surface-2)" strokeWidth="9" strokeLinecap="round" />
      <path
        d={d}
        fill="none"
        stroke="url(#gaugeGrad)"
        strokeWidth="9"
        strokeLinecap="round"
        strokeDasharray={len}
        strokeDashoffset={len * (1 - p)}
      />
    </svg>
  );
}

function CollectionsCard({ rate, collected, outstanding }) {
  const { t } = useLang();
  return (
    <Card padding="lg" className="h-full flex flex-col">
      <CardHeader>
        <div>
          <CardTitle>{t("dash.collections")}</CardTitle>
          <CardDescription>{t("dash.collectionsDesc")}</CardDescription>
        </div>
      </CardHeader>
      <div className="flex-1 flex items-center justify-center">
        <div className="relative w-full max-w-[340px] px-2">
          <HalfGauge value={rate} />
          <div className="absolute inset-x-0 bottom-0 flex flex-col items-center pointer-events-none">
            <span className="font-display text-[46px] font-semibold tabular text-[var(--ink)] leading-none">{rate}%</span>
            <span className="text-sm text-[var(--ink-muted)] mt-1.5">{t("dash.collected")}</span>
          </div>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3 mt-4">
        <GaugeStat dot={T2} label={t("dash.collectedLabel")} value={formatMoney(collected)} />
        <GaugeStat dot="var(--warning)" label={t("dash.outstanding")} value={formatMoney(outstanding)} />
      </div>
    </Card>
  );
}
function GaugeStat({ dot, label, value }) {
  return (
    <div className="rounded-2xl bg-[var(--surface-2)] px-3 py-2.5">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">
        <span className="h-2 w-2 rounded-full" style={{ background: dot }} /> {label}
      </div>
      <div className="text-sm font-semibold text-[var(--ink)] tabular mt-1 truncate">{value}</div>
    </div>
  );
}

/* ─────────────────── Top clients ─────────────────── */
function TopClientsCard({ reports }) {
  const { t } = useLang();
  const clients = reports?.topClients || [];
  const max = Math.max(1, ...clients.map((c) => c.billed));
  return (
    <Card padding="lg" className="h-full">
      <CardHeader>
        <div>
          <CardTitle>{t("dash.topClients")}</CardTitle>
          <CardDescription>{t("dash.byTotalBilled")}</CardDescription>
        </div>
      </CardHeader>
      {!reports ? (
        <ChartSkeleton />
      ) : clients.length ? (
        <div className="space-y-3.5 pt-1">
          {clients.map((c) => (
            <div key={c.id}>
              <div className="flex items-center justify-between text-sm mb-1.5">
                <span className="font-medium text-[var(--ink)] truncate">{c.name}</span>
                <span className="tabular font-semibold text-[var(--ink)] shrink-0 ml-3">{formatMoney(c.billed)}</span>
              </div>
              <div className="h-2 rounded-full bg-[var(--surface-2)] overflow-hidden">
                <div className="h-full rounded-full" style={{ width: `${Math.max(5, (c.billed / max) * 100)}%`, background: `linear-gradient(90deg,${T1},${T3})` }} />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="h-[150px] flex items-center justify-center text-sm text-[var(--ink-muted)]">
          <Users size={16} className="mr-2" /> {t("dash.noClientBilling")}
        </div>
      )}
    </Card>
  );
}

/* ─────────────────── Recent invoices ─────────────────── */
function RecentInvoices({ invoices, onOpen }) {
  const { t } = useLang();
  const rows = invoices || [];
  return (
    <Card padding="lg">
      <CardHeader>
        <div>
          <CardTitle>{t("dash.recentInvoices")}</CardTitle>
          <CardDescription>{t("dash.recentInvoicesDesc")}</CardDescription>
        </div>
      </CardHeader>
      {rows.length === 0 ? (
        <div className="py-10 text-center text-sm text-[var(--ink-muted)]">{t("dash.noInvoices")}</div>
      ) : (
        <div className="flex flex-col divide-y divide-[var(--border)]">
          {rows.map((inv) => (
            <button type="button" key={inv.id} onClick={() => onOpen(inv.id)} className="group flex items-center gap-3 py-3 text-left hover:opacity-90 transition-opacity">
              <div className="h-9 w-9 rounded-full bg-[var(--accent-soft)] text-[var(--accent-strong)] flex items-center justify-center font-semibold text-sm shrink-0">
                {inv.client_name?.[0]?.toUpperCase() || "?"}
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-[var(--ink)] truncate">{inv.client_name || t("common.noClient")}</div>
                <div className="text-xs text-[var(--ink-muted)] tabular">{inv.invoice_number} · {formatDate(inv.issue_date)}</div>
              </div>
              <div className="text-sm font-semibold text-[var(--ink)] tabular shrink-0">{formatMoney(inv.total, inv.currency)}</div>
              <StatusBadge status={inv.effective_status} />
              <ArrowRight size={14} className="text-[var(--ink-muted)] opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />
            </button>
          ))}
        </div>
      )}
    </Card>
  );
}

function ChartSkeleton() {
  return <Skeleton className="h-[150px] rounded-2xl" />;
}

function ChartFallback() {
  return (
    <>
      <Skeleton className="h-[360px] rounded-3xl" />
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5 mt-5">
        <Skeleton className="h-[240px] rounded-3xl" />
        <Skeleton className="h-[240px] rounded-3xl" />
      </div>
    </>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-10 w-48 rounded-2xl" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-[120px] rounded-2xl" />)}
      </div>
      <Skeleton className="h-[92px] rounded-2xl" />
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        <Skeleton className="lg:col-span-8 h-[360px] rounded-3xl" />
        <Skeleton className="lg:col-span-4 h-[360px] rounded-3xl" />
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        <Skeleton className="lg:col-span-4 h-[240px] rounded-3xl" />
        <Skeleton className="lg:col-span-4 h-[240px] rounded-3xl" />
        <Skeleton className="lg:col-span-4 h-[240px] rounded-3xl" />
      </div>
    </div>
  );
}

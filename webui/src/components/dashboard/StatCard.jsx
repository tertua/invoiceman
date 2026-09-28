import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";

function MiniLine({ data, color }) {
  const values = data.map((point) => Number(point?.v) || 0);
  const min = Math.min(...values);
  const range = Math.max(...values) - min || 1;
  const points = values
    .map((value, index) => {
      const x = values.length === 1 ? 55 : (index / (values.length - 1)) * 110;
      const y = 36 - ((value - min) / range) * 30;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <svg viewBox="0 0 110 42" width="110" height="42" aria-hidden="true">
      <polyline points={points} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function MiniBars({ data, color }) {
  const values = data.map((point) => Number(point?.v) || 0);
  const max = Math.max(...values, 1);
  const gap = values.length > 1 ? 4 : 0;
  const width = values.length ? (110 - gap * (values.length - 1)) / values.length : 0;

  return (
    <svg viewBox="0 0 110 42" width="110" height="42" aria-hidden="true">
      {values.map((value, index) => {
        const height = Math.max((value / max) * 30, 2);
        return (
          <rect
            key={index}
            x={index * (width + gap)}
            y={36 - height}
            width={Math.max(width, 1)}
            height={height}
            rx="3"
            fill={color}
          />
        );
      })}
    </svg>
  );
}

export function StatCard({
  label,
  value,
  suffix,
  delta,
  chart = "line",
  data = [],
  icon: Icon,
  accent = false,
}) {
  const positive = delta == null ? null : delta >= 0;
  const color = accent ? "#FFFFFF" : "var(--accent)";
  const ChartCmp = chart === "bars" ? MiniBars : MiniLine;
  const displayValue = value == null || value === "" ? "—" : value;
  const hasData = Array.isArray(data) && data.length > 0;

  return (
    <Card
      variant={accent ? "accent" : "default"}
      className={cn(
        "relative overflow-hidden",
        accent && "text-white"
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2 min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {Icon && (
              <div
                className={cn(
                  "h-7 w-7 rounded-full flex items-center justify-center",
                  accent
                    ? "bg-white/15 text-white"
                    : "bg-[var(--accent-soft)] text-[var(--accent-strong)]"
                )}
              >
                <Icon size={14} />
              </div>
            )}
            <span
              className={cn(
                "text-xs",
                accent ? "text-white/70" : "text-[var(--ink-muted)]"
              )}
            >
              {label}
            </span>
          </div>
          <div className="flex items-baseline gap-1">
            <span className="font-display tabular text-3xl font-semibold tracking-tight">
              {displayValue}
            </span>
            {suffix && (
              <span
                className={cn(
                  "text-sm font-medium",
                  accent ? "text-white/70" : "text-[var(--ink-muted)]"
                )}
              >
                {suffix}
              </span>
            )}
          </div>
          {delta != null && (
            <Badge
              tone={accent ? "ink" : positive ? "success" : "danger"}
              className={cn(accent && "bg-white/15 text-white")}
            >
              {positive ? "+" : ""}
              {delta}%
            </Badge>
          )}
        </div>

        {hasData && (
          <div className="w-[110px] shrink-0 self-end opacity-90">
            <ChartCmp data={data} color={color} />
          </div>
        )}
      </div>
    </Card>
  );
}

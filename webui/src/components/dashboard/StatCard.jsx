import { Card } from "@/components/ui/Card";
import { cn } from "@/lib/utils";
import { StatValue } from "./StatValue";

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
  animateValue = false,
  valueNumber,
  renderValue,
}) {
  const color = accent ? "#FFFFFF" : "var(--accent)";
  const ChartCmp = chart === "bars" ? MiniBars : MiniLine;
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
        <StatValue
          Icon={Icon}
          label={label}
          value={value}
          suffix={suffix}
          delta={delta}
          accent={accent}
          animateValue={animateValue}
          valueNumber={valueNumber}
          renderValue={renderValue}
        />

        {hasData && (
          <div className="w-[110px] shrink-0 self-end opacity-90">
            <ChartCmp data={data} color={color} />
          </div>
        )}
      </div>
    </Card>
  );
}

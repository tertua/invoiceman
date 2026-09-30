import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/Badge";
import { CountUpValue } from "./CountUpValue";

// The label + big value + suffix + delta badge block of a StatCard, extracted
// so StatCard stays under its line budget and the count-up can be toggled
// without touching the card chrome. `renderValue` is the money formatter used
// on the animated path; the static path prints the preformatted string.
export function StatValue({ Icon, label, value, suffix, delta, accent, animateValue, valueNumber, renderValue }) {
  const positive = delta == null ? null : delta >= 0;
  const displayValue = value == null || value === "" ? "—" : value;
  const animated = animateValue && valueNumber != null && renderValue;
  return (
    <div className="space-y-2 min-w-0 flex-1">
      <div className="flex items-center gap-2">
        {Icon && (
          <div
            className={cn(
              "h-7 w-7 rounded-full flex items-center justify-center",
              accent ? "bg-white/15 text-white" : "bg-[var(--accent-soft)] text-[var(--accent-strong)]"
            )}
          >
            <Icon size={14} />
          </div>
        )}
        <span className={cn("text-xs", accent ? "text-white/70" : "text-[var(--ink-muted)]")}>{label}</span>
      </div>
      <div className="flex items-baseline gap-1">
        {animated ? (
          <CountUpValue
            value={valueNumber}
            render={renderValue}
            className="font-display tabular text-3xl font-semibold tracking-tight"
          />
        ) : (
          <span className="font-display tabular text-3xl font-semibold tracking-tight">{displayValue}</span>
        )}
        {suffix && (
          <span className={cn("text-sm font-medium", accent ? "text-white/70" : "text-[var(--ink-muted)]")}>
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
  );
}

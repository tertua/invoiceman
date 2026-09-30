import { motion, useReducedMotion } from "framer-motion";
import { Check } from "lucide-react";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// Three-dot progress rail for the public pay flow: choose -> pay -> done.
// Presentational only — `current` is derived by the page (resolvePayStep), so
// the indicator never owns state and can't drift from the real panel.
const STEPS = [
  { key: "choose", label: "public.stepChoose" },
  { key: "pay", label: "public.stepPay" },
  { key: "done", label: "public.stepDone" },
];

export default function StepIndicator({ current, lang, className }) {
  const reduce = useReducedMotion();
  const activeIndex = Math.max(0, STEPS.findIndex((s) => s.key === current));

  return (
    <ol className={cn("flex items-center gap-1.5", className)} aria-label={t(lang, "public.stepChoose")}>
      {STEPS.map((step, index) => {
        const done = index < activeIndex;
        const active = index === activeIndex;
        return (
          <li key={step.key} className="flex items-center gap-1.5 min-w-0" aria-current={active ? "step" : undefined}>
            <span
              className={cn(
                "h-5 w-5 shrink-0 rounded-full flex items-center justify-center text-[10px] font-semibold transition-colors",
                done
                  ? "bg-[var(--success)] text-white"
                  : active
                    ? "bg-[var(--accent)] text-white"
                    : "bg-[var(--surface-2)] text-[var(--ink-muted)] border border-[var(--border)]"
              )}
            >
              {done ? <Check size={11} strokeWidth={3} /> : index + 1}
            </span>
            <span
              className={cn(
                "text-[11px] font-semibold truncate",
                active ? "text-[var(--ink)]" : "text-[var(--ink-muted)]"
              )}
            >
              {t(lang, step.label)}
            </span>
            {index < STEPS.length - 1 && (
              <motion.span
                aria-hidden="true"
                className="h-[2px] w-4 sm:w-7 rounded-full"
                initial={false}
                animate={{ backgroundColor: done ? "var(--success)" : "var(--border)" }}
                transition={{ duration: reduce ? 0 : 0.25 }}
              />
            )}
          </li>
        );
      })}
    </ol>
  );
}

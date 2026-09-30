import { cn } from "@/lib/utils";

// GatewaySelect renders the project's default-gateway choices. The options are
// the configured providers reported by GET /public/gateway/status (the
// registry), first configured one first — there is no hardcoded provider list.
// An empty registry disables the control instead of inventing a default.
export function GatewaySelect({ value, options, onChange, lang: t }) {
  return (
    <select
      value={value}
      onChange={onChange}
      disabled={!options.length}
      aria-label={t("gateway.gateway")}
      className={cn(
        "h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none transition-colors focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15",
        !options.length && "opacity-60",
      )}
    >
      {options.length ? (
        options.map((name) => (
          <option key={name} value={name}>{name}</option>
        ))
      ) : (
        <option value="">{t("gateway.noneAvailable")}</option>
      )}
    </select>
  );
}

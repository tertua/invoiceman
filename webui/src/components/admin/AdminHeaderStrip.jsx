import { useGatewayStatus } from "@/hooks/useGatewayAdmin";
import { useLang } from "@/context/LangContext";
import { cn } from "@/lib/utils";

// Compact at-a-glance strip for the console header. Reads ONLY
// `useGatewayStatus()` — the same cached `gatewayStatusKey` query the Gateway
// page's status banner already runs — so it never adds a network request.
// Renders nothing while loading or when the list is empty (mirrors
// GatewayStatusBanner behavior).
export function AdminHeaderStrip() {
  const { t } = useLang();
  const { data: gateways = [], isLoading } = useGatewayStatus();
  if (isLoading || !gateways.length) return null;

  const configured = gateways.filter((gw) => gw.configured).length;
  const live = gateways.some((gw) => gw.configured && !gw.sandbox);

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 border-t border-[var(--border)]/60 px-1 py-2 text-xs text-[var(--ink-muted)]">
      <span className="flex items-center gap-1.5 font-medium text-[var(--ink)]">
        <span className="h-1.5 w-1.5 rounded-full bg-[var(--accent)]" />
        {t("admin.stripConfigured", { count: configured })}
      </span>
      <span className="flex flex-wrap items-center gap-1.5">
        {gateways.map((gw) => (
          <span
            key={gw.name}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-semibold",
              gw.configured
                ? "bg-[var(--success)]/12 text-[var(--success)]"
                : "bg-[var(--surface-2)] text-[var(--ink-muted)]",
            )}
            title={gw.configured ? t("gateway.configured") : t("gateway.notConfigured")}
          >
            <code>{gw.name}</code>
            {gw.configured && (
              <span className="opacity-70">{gw.sandbox ? t("admin.stripSandbox") : t("admin.stripLive")}</span>
            )}
          </span>
        ))}
      </span>
      <span className="rounded-full bg-[var(--surface-2)] px-2 py-0.5 text-[11px] font-semibold">
        {live ? t("admin.stripLive") : t("admin.stripSandbox")}
      </span>
    </div>
  );
}

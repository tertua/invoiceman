import { Network } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { StatusPill } from "@/components/gateway/StatusPill";
import { useLang } from "@/context/LangContext";
import { useGatewayStatus } from "@/hooks/useGatewayAdmin";

export function AdminGatewayStatusBanner() {
  const { t } = useLang();
  const { data: gateways = [], isLoading } = useGatewayStatus();
  if (isLoading || !gateways.length) return null;
  return (
    <Card className="mb-6">
      <div className="mb-3 flex items-center gap-2">
        <Network size={16} className="text-[var(--accent-strong)]" />
        <h2 className="font-display text-base font-semibold">{t("gateway.availability")}</h2>
      </div>
      <div className="flex flex-wrap gap-2">
        {gateways.map((gw) => (
          <span
            key={gw.name}
            className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] px-3 py-1.5 text-xs font-semibold"
            title={gw.configured ? t("gateway.configured") : t("gateway.notConfigured")}
          >
            <code>{gw.name}</code>
            <StatusPill active={gw.configured} t={t} />
            {gw.configured && (
              <span className="text-[var(--ink-muted)]">
                {gw.sandbox ? t("gateway.sandbox") : t("gateway.live")}
              </span>
            )}
          </span>
        ))}
      </div>
    </Card>
  );
}

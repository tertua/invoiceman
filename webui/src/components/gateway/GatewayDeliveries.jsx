import { RefreshCw, Webhook } from "lucide-react";
import { StatusPill } from "@/components/gateway/StatusPill";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { QueryError } from "@/components/ui/QueryError";
import { useLang } from "@/context/LangContext";
import { useGatewayDeliveries, useRetryGatewayDelivery } from "@/hooks/useGatewayAdmin";

export function GatewayDeliveries() {
  const { t } = useLang();
  const { data: deliveries = [], error } = useGatewayDeliveries();
  const retry = useRetryGatewayDelivery();
  if (error && error.status !== 401) return <QueryError error={error} />;
  return (
    <Card padding="none" className="mt-6 overflow-hidden">
      <div className="flex items-center gap-2 border-b border-[var(--border)] px-4 py-3">
        <Webhook size={16} className="text-[var(--accent-strong)]" />
        <h2 className="font-display font-semibold">{t("admin.sectionDeliveries")}</h2>
      </div>
      <div className="max-h-[360px] overflow-auto">
        {deliveries.length ? (
          deliveries.map((item) => (
            <div key={item.id} className="border-b border-[var(--border)] px-4 py-2.5 transition-colors last:border-0 hover:bg-[var(--surface-2)]/60">
              <div className="flex items-center justify-between gap-3">
                <code className="truncate text-xs text-[var(--ink)]">{item.order_id}</code>
                <StatusPill status={item.status} t={t} />
              </div>
              <div className="mt-1 flex items-center justify-between gap-2 text-xs text-[var(--ink-muted)]">
                <span className="tabular">{item.gateway} · {t("gateway.attempt")} {item.attempt} · HTTP {item.resp_code || "-"}</span>
                {item.status !== "delivered" && (
                  <Button size="iconSm" variant="ghost" onClick={() => retry.mutate(item.id)} disabled={retry.isPending} aria-label={t("gateway.retry")}>
                    <RefreshCw size={13} />
                  </Button>
                )}
              </div>
            </div>
          ))
        ) : (
          <p className="p-6 text-sm text-[var(--ink-muted)]">{t("gateway.noDeliveries")}</p>
        )}
      </div>
    </Card>
  );
}

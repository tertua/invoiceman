import { PageHeader } from "@/components/layout/PageHeader";
import { SettlementPanel } from "@/components/settlement/SettlementPanel";
import { useLang } from "@/context/LangContext";

export default function AdminSettlement() {
  const { t } = useLang();
  return (
    <div>
      <PageHeader title={t("gateway.settlement")} description={t("gateway.settlementDesc")} />
      <SettlementPanel />
    </div>
  );
}

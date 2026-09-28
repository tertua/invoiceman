import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/api/admin";
import { gatewaySettlementApi } from "@/api/gatewaySettlement";
import { gatewayTransactionsKey } from "@/hooks/useGatewayAdmin";

export function useGatewaySettlement() {
  return useQuery({
    queryKey: ["admin", "gateway", "settlement"],
    queryFn: gatewaySettlementApi.summary,
  });
}

export function useGatewayTransactionsPage(page, perPage = 20) {
  return useQuery({
    queryKey: [...gatewayTransactionsKey, { page, perPage }],
    queryFn: () => adminApi.listGatewayTransactions({ page, per_page: perPage }),
    placeholderData: (previous) => previous,
  });
}

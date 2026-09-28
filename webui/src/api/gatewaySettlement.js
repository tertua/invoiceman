import { apiClient } from "./http";

export const gatewaySettlementApi = {
  summary: () => apiClient.get("/admin/gateway/settlement").then((r) => r.data.summary),
};

import { apiClient } from "./http";

export const gatewayApi = {
  config: () => apiClient.get("/public/gateway/config").then((r) => r.data),
  status: () => apiClient.get("/public/gateway/status").then((r) => r.data.gateways),
  createInvoiceIntent: (invoiceId) =>
    apiClient.post(`/invoices/${invoiceId}/intents`).then((r) => r.data),
};

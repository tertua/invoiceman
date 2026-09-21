import { apiClient } from "./client";

export const gatewayApi = {
  config: () => apiClient.get("/public/gateway/config").then((r) => r.data),
  createInvoiceIntent: (invoiceId) =>
    apiClient.post("/gateway/invoice-intents", { invoiceId }).then((r) => r.data),
};

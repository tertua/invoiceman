import { apiClient } from "./http";

export const paymentsApi = {
  list: () => apiClient.get("/payments").then((r) => r.data),
  create: (payload) => apiClient.post("/payments", payload).then((r) => r.data.payment),
  remove: (id) => apiClient.delete(`/payments/${id}`).then(() => undefined),
  createOnlineLink: (invoiceId) => apiClient.post("/payments/online", { invoiceId }).then((r) => r.data),
  sendOnlineLink: (invoiceId, email) => apiClient.post("/payments/online/send", { invoiceId, email }).then((r) => r.data),
};

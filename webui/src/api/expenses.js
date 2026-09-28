import { apiClient } from "./http";

export const expensesApi = {
  list: (params = {}) => apiClient.get("/expenses", { params }).then((r) => r.data),
  create: (payload) => apiClient.post("/expenses", payload).then((r) => r.data.expense),
  update: (id, payload) => apiClient.patch(`/expenses/${id}`, payload).then((r) => r.data.expense),
  remove: (id) => apiClient.delete(`/expenses/${id}`).then(() => undefined),
  uploadReceipt: (id, file) => {
    const form = new FormData();
    form.append("file", file);
    return apiClient.post(`/expenses/${id}/receipt`, form).then((r) => r.data.expense);
  },
  deleteReceipt: (id) => apiClient.delete(`/expenses/${id}/receipt`).then(() => undefined),
};

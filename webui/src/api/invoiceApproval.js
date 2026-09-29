import { apiClient } from "./http";

// Approval workflow (D11): submit by any member, approve/reject owner-only.
export const invoiceApprovalApi = {
  submit: (id) => apiClient.post(`/invoices/${id}/submit`).then((r) => r.data.invoice),
  approve: (id) => apiClient.post(`/invoices/${id}/approve`).then((r) => r.data.invoice),
  reject: (id) => apiClient.post(`/invoices/${id}/reject`).then((r) => r.data.invoice),
};

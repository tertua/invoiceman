import { apiClient } from "./http";

// Counts per invoice status for the list tabs. Kept out of api/invoices.js so
// that baseline-locked seam file never grows.
export const invoiceStatusCountsApi = {
  get: () => apiClient.get("/invoices/status-counts").then((r) => r.data.counts),
};

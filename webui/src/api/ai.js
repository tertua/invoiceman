import { apiClient } from "./http";

export function isAiUnavailable(error) {
  return error?.status === 501;
}

export function isAiFailure(error) {
  return error?.status === 502;
}

export function isAiRateLimited(error) {
  return error?.status === 429;
}

export const aiApi = {
  receiptParse: (file) => {
    const form = new FormData();
    form.append("file", file);
    return apiClient
      .post("/ai/receipt-parse", form, {
        headers: { "Content-Type": "multipart/form-data" },
      })
      .then((r) => r.data.result);
  },
  businessSummary: () => apiClient.post("/ai/business-summary").then((r) => r.data),
  paymentReminder: (invoiceId, tone) =>
    apiClient.post("/ai/payment-reminder", { invoiceId, tone }).then((r) => r.data),
  writeNote: (payload) => apiClient.post("/ai/write-note", payload).then((r) => r.data.text),
};

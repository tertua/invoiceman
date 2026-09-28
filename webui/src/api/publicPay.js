import { apiClient } from "./http";

// Dev path (api seam): [done] createTransaction(token, method, extra) — payment_method for the picker, pay_currency for the crypto widget; get(token, payCurrency) re-reads the public list with ?pay_currency= so the backend gates crypto on that asset's live minimum instead of a default one; [later] `extra` stays the open slot for network fields, callers never change. Shared shape: PublicPay.jsx + CryptoWidget.jsx + useCryptoPayable.js. End dev path
export const publicPayApi = {
  get: (token, payCurrency) => apiClient.get(`/public/pay/${token}`, payCurrency ? { params: { pay_currency: payCurrency } } : undefined).then((r) => r.data),
  createTransaction: (token, method, extra) => apiClient.post(`/public/pay/${token}/transaction`, { ...(method ? { payment_method: method } : {}), ...(extra || {}) }).then((r) => r.data),
  status: (token) => apiClient.get(`/public/pay/${token}/status`).then((r) => r.data),
};

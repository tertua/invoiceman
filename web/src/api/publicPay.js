import { apiClient } from "./http";

export const publicPayApi = {
  get: (token) => apiClient.get(`/public/pay/${token}`).then((r) => r.data),
  createTransaction: (token, method, extra) => apiClient.post(`/public/pay/${token}/transaction`, { ...(method ? { payment_method: method } : {}), ...(extra || {}) }).then((r) => r.data),
  status: (token) => apiClient.get(`/public/pay/${token}/status`).then((r) => r.data),
};

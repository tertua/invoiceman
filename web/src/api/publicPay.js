import { apiClient } from "./client";

export const publicPayApi = {
  get: (token) => apiClient.get(`/public/pay/${token}`).then((r) => r.data),
  createTransaction: (token) =>
    apiClient.post(`/public/pay/${token}/transaction`).then((r) => r.data),
  status: (token) => apiClient.get(`/public/pay/${token}/status`).then((r) => r.data),
};

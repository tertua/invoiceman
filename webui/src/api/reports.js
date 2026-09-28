import { apiClient } from "./http";

export const reportsApi = {
  get: (params = {}) => apiClient.get("/reports", { params }).then((r) => r.data),
};

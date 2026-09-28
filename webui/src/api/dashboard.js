import { apiClient } from "./http";

export const dashboardApi = {
  get: (params = {}) => apiClient.get("/dashboard", { params }).then((r) => r.data),
};

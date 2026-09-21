import { apiClient } from "./client";

export const dashboardApi = {
  get: (params = {}) => apiClient.get("/dashboard", { params }).then((r) => r.data),
};

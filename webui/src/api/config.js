import { apiClient } from "./http";

export const configApi = {
  get: () => apiClient.get("/config").then((r) => r.data),
};

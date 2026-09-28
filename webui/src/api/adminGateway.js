import { apiClient } from "./http";

export const adminGatewayApi = {
  deleteGatewayProject: (slug) =>
    apiClient.delete(`/admin/gateway/projects/${slug}`).then((r) => r.data),
};

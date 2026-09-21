import { apiClient } from "./client";

export const adminApi = {
  listUsers: () => apiClient.get("/admin/users").then((r) => r.data.users),
  updateUserRole: (id, role) =>
    apiClient.patch(`/admin/users/${id}/role`, { role }).then((r) => r.data.user),
};

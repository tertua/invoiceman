import { apiClient } from "./http";

export const itemsApi = {
  list: () => apiClient.get("/items").then((r) => r.data.items),
  create: (payload) => apiClient.post("/items", payload).then((r) => r.data.item),
  update: (id, payload) => apiClient.patch(`/items/${id}`, payload).then((r) => r.data.item),
  remove: (id) => apiClient.delete(`/items/${id}`).then(() => undefined),
};

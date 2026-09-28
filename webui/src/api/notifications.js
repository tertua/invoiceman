import { apiClient } from "./http";

export const notificationsApi = {
  listEndpoints: () => apiClient.get("/notifications/endpoints").then((r) => r.data.endpoints),
  createEndpoint: (payload) =>
    apiClient.post("/notifications/endpoints", payload).then((r) => r.data.endpoint),
  updateEndpoint: (id, payload) =>
    apiClient.patch(`/notifications/endpoints/${id}`, payload).then((r) => r.data.endpoint),
  deleteEndpoint: (id) => apiClient.delete(`/notifications/endpoints/${id}`),
  rotateSecret: (id) =>
    apiClient.post(`/notifications/endpoints/${id}/rotate-secret`).then((r) => r.data.endpoint),
  testEndpoint: (id) =>
    apiClient.post(`/notifications/endpoints/${id}/test`).then((r) => r.data),
  listDeliveries: (params = {}) =>
    apiClient.get("/notifications/deliveries", { params }).then((r) => r.data.deliveries),
  retryDelivery: (id) =>
    apiClient.post(`/notifications/deliveries/${id}/retry`).then((r) => r.data.delivery),
};

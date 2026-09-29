import { apiClient } from "./http";

// Org endpoints reuse the shared client (baseURL /api/v1); each call unwraps its primary key.
export const orgsApi = {
  me: () => apiClient.get("/orgs/me").then((r) => r.data.org),
  members: () => apiClient.get("/orgs/members").then((r) => r.data.members),
  invites: () => apiClient.get("/orgs/invites").then((r) => r.data.invites),
  invite: (payload) => apiClient.post("/orgs/invites", payload).then((r) => r.data.invite),
  revoke: (id) => apiClient.delete(`/orgs/invites/${id}`).then(() => undefined),
  accept: (payload) => apiClient.post("/orgs/invites/accept", payload).then((r) => r.data.org),
  activate: (id) => apiClient.post(`/orgs/${id}/activate`).then((r) => r.data.org),
  rename: (id, payload) => apiClient.patch(`/orgs/${id}`, payload).then((r) => r.data.org),
};

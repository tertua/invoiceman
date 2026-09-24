import { apiClient } from "./http";

export const settingsApi = {
  get: () => apiClient.get("/settings").then((r) => r.data.settings),
  update: (payload) => apiClient.patch("/settings", payload).then((r) => r.data.settings),
  methods: () => apiClient.get("/settings/methods").then((r) => r.data.methods),
  uploadLogo: (file) => {
    const form = new FormData();
    form.append("logo", file);
    return apiClient.post("/settings/logo", form).then((r) => r.data.settings);
  },
};
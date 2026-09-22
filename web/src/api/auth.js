import { apiClient } from "./http";

const captchaHeaders = (token) => (token ? { "X-Captcha-Token": token } : {});

export const authApi = {
  register: (payload, captchaToken) =>
    apiClient.post("/auth/register", payload, { headers: captchaHeaders(captchaToken) }).then((r) => r.data),
  login: (payload, captchaToken) =>
    apiClient.post("/auth/login", payload, { headers: captchaHeaders(captchaToken) }).then((r) => r.data),
  logout: () => apiClient.post("/auth/logout").then(() => undefined),
  me: () => apiClient.get("/auth/me").then((r) => r.data),
  updateProfile: (payload) => apiClient.patch("/auth/profile", payload).then((r) => r.data),
  changePassword: (payload) => apiClient.patch("/auth/password", payload).then((r) => r.data),
  forgotPassword: (payload, captchaToken) =>
    apiClient.post("/auth/forgot-password", payload, { headers: captchaHeaders(captchaToken) }).then((r) => r.data),
  resetPassword: (payload) => apiClient.post("/auth/reset-password", payload).then((r) => r.data),
};

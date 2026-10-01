import { apiClient } from "./http";

const captchaHeaders = (token) => (token ? { "X-Captcha-Token": token } : {});

// authVerifyApi holds the email-verification endpoints, kept out of auth.js so
// that file stays within its size ratchet.
export const authVerifyApi = {
  verifyEmail: (payload) => apiClient.post("/auth/verify-email", payload).then((r) => r.data),
  resendVerification: (payload, captchaToken) =>
    apiClient.post("/auth/verify-email/resend", payload, { headers: captchaHeaders(captchaToken) }).then((r) => r.data),
};

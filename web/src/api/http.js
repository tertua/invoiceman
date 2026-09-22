import axios from "axios";
import { localizeApiError } from "@/lib/utils";

export const apiClient = axios.create({
  baseURL: "/api/v1",
  withCredentials: true,
  headers: { "Content-Type": "application/json" },
});

// CSRF double-submit: mirror the csrf_token cookie as its header on every
// request. The backend only enforces it for session-cookie mutations;
// public and API-key calls are unaffected.
function csrfToken() {
  try {
    const m = document.cookie.match(/(?:^|; )csrf_token=([^;]*)/);
    return m ? decodeURIComponent(m[1]) : "";
  } catch {
    return "";
  }
}

apiClient.interceptors.request.use((config) => {
  const token = csrfToken();
  if (token) {
    config.headers = config.headers ?? {};
    config.headers["X-CSRF-Token"] = token;
  }
  return config;
});

apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    const message = localizeApiError(
      err.response?.data?.error?.message ||
        err.message ||
        "Request failed"
    );
    return Promise.reject({
      status: err.response?.status,
      message,
      details: err.response?.data?.error?.details,
      original: err,
    });
  }
);

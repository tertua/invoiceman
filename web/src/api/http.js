import axios from "axios";
import { localizeApiError } from "@/lib/utils";

// Broadcast when an authenticated call fails with 401 so the app can
// drop the stale user and bounce to /login without a manual reload.
// AuthContext listens for this; ProtectedShell then redirects.
export const AUTH_EXPIRED_EVENT = "invoiceman:auth-expired";

// 401s that are part of a normal unauthenticated flow — they must stay
// inline (e.g. wrong password) and never trigger a global logout.
const AUTH_EXPIRED_EXCLUSIONS = [
  "/auth/login",
  "/auth/register",
  "/auth/forgot-password",
  "/auth/reset-password",
  "/auth/me",
  "/auth/logout",
  "/config",
  "/public/",
];

function shouldBroadcastExpired(url) {
  if (typeof url !== "string" || !url) return true;
  return !AUTH_EXPIRED_EXCLUSIONS.some((excluded) => url.includes(excluded));
}

export function broadcastAuthExpired() {
  try {
    window.dispatchEvent(new CustomEvent(AUTH_EXPIRED_EVENT));
  } catch {
    /* non-browser / dispatch unsupported — ProtectedShell still guards routes */
  }
}

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
    const status = err.response?.status;
    if (status === 401 && shouldBroadcastExpired(err.config?.url)) {
      broadcastAuthExpired();
    }
    const message = localizeApiError(
      err.response?.data?.error?.message ||
        err.message ||
        "Request failed"
    );
    return Promise.reject({
      status,
      message,
      details: err.response?.data?.error?.details,
      original: err,
    });
  }
);

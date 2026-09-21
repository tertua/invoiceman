import axios from "axios";
import { localizeApiError } from "@/lib/utils";

export const apiClient = axios.create({
  baseURL: "/api",
  withCredentials: true,
  headers: { "Content-Type": "application/json" },
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

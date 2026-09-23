import { createContext, useContext, useEffect, useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "@/api/auth";
import { AUTH_EXPIRED_EVENT } from "@/api/http";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [sessionExpired, setSessionExpired] = useState(false);
  const queryClient = useQueryClient();

  const refresh = useCallback(async () => {
    try {
      const { user } = await authApi.me();
      setUser(user);
      setSessionExpired(false);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  // Any authenticated API call returning 401 means the session is gone
  // (expired, revoked, or wiped by a server restart). Drop the stale user
  // so ProtectedShell bounces to /login — no manual reload needed.
  useEffect(() => {
    const onExpired = () => {
      setUser(null);
      setSessionExpired(true);
      queryClient.clear();
    };
    window.addEventListener(AUTH_EXPIRED_EVENT, onExpired);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, onExpired);
  }, [queryClient]);

  const login = useCallback(async (credentials, captchaToken) => {
    queryClient.clear(); // drop previous account's cached queries before swapping identity
    const { user } = await authApi.login(credentials, captchaToken);
    setUser(user);
    setSessionExpired(false);
    return user;
  }, [queryClient]);

  const register = useCallback(async (payload, captchaToken) => {
    const { user } = await authApi.register(payload, captchaToken);
    setUser(user);
    setSessionExpired(false);
    return user;
  }, []);

  const updateProfile = useCallback(async (payload) => {
    const { user } = await authApi.updateProfile(payload);
    setUser(user);
    return user;
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      setUser(null);
      setSessionExpired(false);
      queryClient.clear();
    }
  }, [queryClient]);

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout, refresh, updateProfile, sessionExpired, setSessionExpired }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

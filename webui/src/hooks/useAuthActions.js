import { useCallback, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "@/api/auth";
import { AUTH_EXPIRED_EVENT } from "@/api/http";

// Auth actions + the 401-expiry listener, lifted out of AuthProvider; the setters come from its state (activeOrg lands here later).
export function useAuthActions({ setUser, setLoading, setSessionExpired }) {
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
  }, [setUser, setLoading, setSessionExpired]);

  // A 401 on any authenticated call means the session is gone — drop the stale user so ProtectedShell bounces to /login.
  useEffect(() => {
    const onExpired = () => {
      setUser(null);
      setSessionExpired(true);
      queryClient.clear();
    };
    window.addEventListener(AUTH_EXPIRED_EVENT, onExpired);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, onExpired);
  }, [queryClient, setUser, setSessionExpired]);

  const login = useCallback(async (credentials, captchaToken) => {
    queryClient.clear(); // drop previous account's cached queries before swapping identity
    const { user } = await authApi.login(credentials, captchaToken);
    setUser(user);
    setSessionExpired(false);
    return user;
  }, [queryClient, setUser, setSessionExpired]);

  const register = useCallback(async (payload, captchaToken) => {
    queryClient.clear(); // drop previous account's cached queries before swapping identity
    const { user } = await authApi.register(payload, captchaToken);
    setUser(user);
    setSessionExpired(false);
    return user;
  }, [queryClient, setUser, setSessionExpired]);

  const updateProfile = useCallback(async (payload) => {
    const { user } = await authApi.updateProfile(payload);
    setUser(user);
    return user;
  }, [setUser]);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      setUser(null);
      setSessionExpired(false);
      queryClient.clear();
    }
  }, [queryClient, setUser, setSessionExpired]);

  return { refresh, login, register, logout, updateProfile };
}

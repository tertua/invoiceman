import { useQuery } from "@tanstack/react-query";
import { configApi } from "@/api/config";

export const appConfigKey = ["app-config"];

export function useAppConfig() {
  return useQuery({ queryKey: appConfigKey, queryFn: () => configApi.get(), staleTime: Infinity });
}

export function useAppName() {
  return useAppConfig().data?.appName || "TuPay";
}

export function useAllowRegistration() {
  return useAppConfig().data?.allowRegistration ?? true;
}

export function useOidcEnabled() {
  return useAppConfig().data?.oidcEnabled ?? false;
}

import { useQuery } from "@tanstack/react-query";
import { configApi } from "@/api/config";

export const appConfigKey = ["app-config"];

export function useAppConfig() {
  return useQuery({
    queryKey: appConfigKey,
    queryFn: () => configApi.get(),
    staleTime: Infinity,
  });
}

export function useAppName() {
  const { data } = useAppConfig();
  return data?.appName || "Invoiceman";
}

export function useAllowRegistration() {
  const { data } = useAppConfig();
  return data?.allowRegistration ?? true;
}

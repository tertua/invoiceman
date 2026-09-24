import { useQuery } from "@tanstack/react-query";
import { settingsApi } from "@/api/settings";

export const settingsMethodsKey = ["settings", "methods"];

// useMidtransMethods lists the provider-neutral methods an owner can enable
// for Midtrans. The catalog is static, so it is cached for the session.
export function useMidtransMethods() {
  return useQuery({
    queryKey: settingsMethodsKey,
    queryFn: () => settingsApi.methods(),
    staleTime: Infinity,
  });
}

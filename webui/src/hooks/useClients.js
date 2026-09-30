import { useQuery } from "@tanstack/react-query";
import { clientsApi } from "@/api/clients";

export const clientsKey = ["clients"];
export const clientKey = (id) => ["client", id];

export function useClients() {
  return useQuery({ queryKey: clientsKey, queryFn: () => clientsApi.list() });
}

export function useClient(id) {
  return useQuery({
    queryKey: clientKey(id),
    queryFn: () => clientsApi.get(id),
    enabled: !!id,
  });
}

// Mutation hooks live in useClientMutations.js (file-size split); re-exported
// here so callers keep importing from "@/hooks/useClients".
export { useCreateClient, useUpdateClient, useDeleteClient } from "./useClientMutations";

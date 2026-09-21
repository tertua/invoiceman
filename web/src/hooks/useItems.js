import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { itemsApi } from "@/api/items";

export const itemsKey = ["items"];

export function useItems() {
  return useQuery({ queryKey: itemsKey, queryFn: () => itemsApi.list() });
}

export function useItemMutations() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: itemsKey });
  return {
    create: useMutation({ mutationFn: itemsApi.create, onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, payload }) => itemsApi.update(id, payload), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: itemsApi.remove, onSuccess: invalidate }),
  };
}

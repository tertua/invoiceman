import { useQuery } from "@tanstack/react-query";
import { itemsApi } from "@/api/items";

export const itemsKey = ["items"];

export function useItems() {
  return useQuery({ queryKey: itemsKey, queryFn: () => itemsApi.list() });
}

// Mutation hooks live in useItemMutations.js (file-size split); re-exported
// here so callers keep importing from "@/hooks/useItems".
export { useItemMutations } from "./useItemMutations";

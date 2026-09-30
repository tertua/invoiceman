import { useQuery } from "@tanstack/react-query";
import { expensesApi } from "@/api/expenses";

export const expensesKey = (params) => ["expenses", params || {}];

export function useExpenses(params) {
  return useQuery({ queryKey: expensesKey(params), queryFn: () => expensesApi.list(params) });
}

// Mutation hooks live in useExpenseMutations.js (file-size split); re-exported
// here so callers keep importing from "@/hooks/useExpenses".
export { useExpenseMutations } from "./useExpenseMutations";

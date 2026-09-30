import { useQuery } from "@tanstack/react-query";
import { paymentsApi } from "@/api/payments";

export const paymentsKey = ["payments"];

export function usePayments() {
  return useQuery({ queryKey: paymentsKey, queryFn: () => paymentsApi.list() });
}

// Mutation hooks live in usePaymentMutations.js (file-size split); re-exported
// here so callers keep importing from "@/hooks/usePayments".
export { usePaymentMutations } from "./usePaymentMutations";

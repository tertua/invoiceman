import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { paymentsApi } from "@/api/payments";

export const paymentsKey = ["payments"];

export function usePayments() {
  return useQuery({ queryKey: paymentsKey, queryFn: () => paymentsApi.list() });
}

export function usePaymentMutations() {
  const qc = useQueryClient();
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: paymentsKey });
    qc.invalidateQueries({ queryKey: ["invoices"] });
    qc.invalidateQueries({ queryKey: ["invoice"] });
    qc.invalidateQueries({ queryKey: ["dashboard"] });
    qc.invalidateQueries({ queryKey: ["reports"] });
  };
  return {
    create: useMutation({ mutationFn: paymentsApi.create, onSuccess: invalidate }),
    remove: useMutation({ mutationFn: paymentsApi.remove, onSuccess: invalidate }),
  };
}

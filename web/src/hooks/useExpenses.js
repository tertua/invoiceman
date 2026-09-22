import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { expensesApi } from "@/api/expenses";

export const expensesKey = (params) => ["expenses", params || {}];

export function useExpenses(params) {
  return useQuery({ queryKey: expensesKey(params), queryFn: () => expensesApi.list(params) });
}

export function useExpenseMutations() {
  const qc = useQueryClient();
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["expenses"] });
    qc.invalidateQueries({ queryKey: ["reports"] });
  };
  return {
    create: useMutation({ mutationFn: expensesApi.create, onSuccess: invalidate }),
    update: useMutation({ mutationFn: ({ id, payload }) => expensesApi.update(id, payload), onSuccess: invalidate }),
    remove: useMutation({ mutationFn: expensesApi.remove, onSuccess: invalidate }),
    uploadReceipt: useMutation({ mutationFn: ({ id, file }) => expensesApi.uploadReceipt(id, file), onSuccess: invalidate }),
    deleteReceipt: useMutation({ mutationFn: (id) => expensesApi.deleteReceipt(id), onSuccess: invalidate }),
  };
}

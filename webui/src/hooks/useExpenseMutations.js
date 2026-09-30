import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { expensesApi } from "@/api/expenses";
import { useLang } from "@/context/LangContext";
import { restoreSnapshots, snapshotQueries } from "@/lib/optimistic";

// Expenses cache is { expenses, categories, totals }; totals are recomputed by
// the invalidation on settle, so optimistic patches only touch the list. Keys
// are param-filtered, so we snapshot/patch every variant under ["expenses"].
function invalidateAll(qc) {
  qc.invalidateQueries({ queryKey: ["expenses"] });
  qc.invalidateQueries({ queryKey: ["reports"] });
}

function patchExpenses(qc, mapFn) {
  qc.setQueriesData({ queryKey: ["expenses"] }, (old) =>
    old && Array.isArray(old.expenses) ? { ...old, expenses: mapFn(old.expenses) } : old,
  );
}

export function useExpenseMutations() {
  const qc = useQueryClient();
  const { t } = useLang();
  return {
    // No client-side id to insert, so create stays invalidate-only.
    create: useMutation({ mutationFn: expensesApi.create, onSuccess: invalidateAll }),
    // ExpenseModal catches inline, so update is silent.
    update: useMutation({
      mutationFn: ({ id, payload }) => expensesApi.update(id, payload),
      onMutate: async ({ id, payload }) => {
        const prev = await snapshotQueries(qc, [["expenses"]]);
        patchExpenses(qc, (list) => list.map((e) => (e.id === id ? { ...e, ...payload } : e)));
        return { prev };
      },
      onError: (_err, _vars, ctx) => {
        if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
      },
      onSettled: invalidateAll,
    }),
    // Expenses list deletes without a catch, so the hook owns the toast.
    remove: useMutation({
      mutationFn: expensesApi.remove,
      onMutate: async (id) => {
        const prev = await snapshotQueries(qc, [["expenses"]]);
        patchExpenses(qc, (list) => list.filter((e) => e.id !== id));
        return { prev };
      },
      onError: (_err, id, ctx) => {
        if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
        toast.error(t("expenses.deleteFailed"), { id: `expense-delete-${id}` });
      },
      onSettled: invalidateAll,
    }),
    uploadReceipt: useMutation({
      mutationFn: ({ id, file }) => expensesApi.uploadReceipt(id, file),
      onSuccess: invalidateAll,
    }),
    deleteReceipt: useMutation({ mutationFn: (id) => expensesApi.deleteReceipt(id), onSuccess: invalidateAll }),
  };
}

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { paymentsApi } from "@/api/payments";
import { useLang } from "@/context/LangContext";
import { paymentsKey } from "./usePayments";
import { restoreSnapshots, snapshotQueries } from "@/lib/optimistic";

// Payments cache is { payments, totals }; totals are recomputed by the
// invalidation on settle, so the optimistic step only touches the list.
function invalidateAll(qc) {
  qc.invalidateQueries({ queryKey: paymentsKey });
  qc.invalidateQueries({ queryKey: ["invoices"] });
  qc.invalidateQueries({ queryKey: ["invoice"] });
  qc.invalidateQueries({ queryKey: ["dashboard"] });
  qc.invalidateQueries({ queryKey: ["reports"] });
}

export function usePaymentMutations() {
  const qc = useQueryClient();
  const { t } = useLang();
  return {
    // Create has no client-side id to insert, so it stays invalidate-only.
    create: useMutation({
      mutationFn: ({ payload, key }) => paymentsApi.create(payload, key),
      onSuccess: invalidateAll,
    }),
    // Voiding drops the row instantly; callers (Payments/InvoicePaymentCard) have
    // no error UI, so the hook owns the toast and rolls back on failure.
    remove: useMutation({
      mutationFn: ({ id, reason }) => paymentsApi.remove(id, reason),
      onMutate: async ({ id }) => {
        const prev = await snapshotQueries(qc, [paymentsKey]);
        qc.setQueryData(paymentsKey, (old) =>
          old ? { ...old, payments: (old.payments || []).filter((p) => p.id !== id) } : old,
        );
        return { prev };
      },
      onError: (_err, { id }, ctx) => {
        if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
        toast.error(t("payments.voidFailed"), { id: `payment-void-${id}` });
      },
      onSettled: invalidateAll,
    }),
  };
}

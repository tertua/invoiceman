import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { invoicesApi } from "@/api/invoices";
import { useLang } from "@/context/LangContext";
import { invoiceKey } from "./useInvoices";
import { removeFromList, replaceInList, restoreSnapshots, snapshotQueries } from "@/lib/optimistic";

// Every invoice mutation re-syncs these aggregates (dashboard/client totals may
// change with status or deletion).
function invalidateAll(qc) {
  qc.invalidateQueries({ queryKey: ["invoices"] });
  qc.invalidateQueries({ queryKey: ["dashboard"] });
  qc.invalidateQueries({ queryKey: ["clients"] });
}

export function useCreateInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload) => invoicesApi.create(payload),
    onSuccess: () => invalidateAll(qc),
  });
}

export function useUpdateInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }) => invoicesApi.update(id, payload),
    onSuccess: (inv) => {
      invalidateAll(qc);
      if (inv?.id) qc.invalidateQueries({ queryKey: invoiceKey(inv.id) });
    },
  });
}

// Optimistic status change: patch every cached list variant + the detail, roll
// back on error, re-sync on settle. Status is the highest-traffic mutation and
// its caller (InvoiceDetail) has no error UI, so the hook owns the toast.
export function useSetInvoiceStatus() {
  const qc = useQueryClient();
  const { t } = useLang();
  return useMutation({
    mutationFn: ({ id, status }) => invoicesApi.setStatus(id, status),
    onMutate: async ({ id, status }) => {
      const prev = await snapshotQueries(qc, [["invoices"], invoiceKey(id)]);
      replaceInList(qc, ["invoices"], (inv) => inv.id === id, { status, effective_status: status });
      qc.setQueryData(invoiceKey(id), (old) => (old ? { ...old, status, effective_status: status } : old));
      return { prev };
    },
    onError: (_err, { id }, ctx) => {
      if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
      toast.error(t("invoices.statusFailed"), { id: `invoice-status-${id}` });
    },
    onSettled: (inv, _err, vars) => {
      invalidateAll(qc);
      qc.invalidateQueries({ queryKey: invoiceKey(inv?.id || vars.id) });
    },
  });
}

// Optimistic delete: drop the row from every cached list variant, restore on
// error, re-sync on settle. InvoiceDetail catches silently (banner); the toast
// is deduped by id so both surfaces do not stack.
export function useDeleteInvoice() {
  const qc = useQueryClient();
  const { t } = useLang();
  return useMutation({
    mutationFn: (id) => invoicesApi.remove(id),
    onMutate: async (id) => {
      const prev = await snapshotQueries(qc, [["invoices"], invoiceKey(id)]);
      removeFromList(qc, ["invoices"], (inv) => inv.id === id);
      return { prev };
    },
    onError: (_err, id, ctx) => {
      if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
      toast.error(t("invoices.deleteFailed"), { id: `invoice-delete-${id}` });
    },
    onSettled: () => {
      invalidateAll(qc);
    },
  });
}

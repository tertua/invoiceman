import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { clientsApi } from "@/api/clients";
import { useLang } from "@/context/LangContext";
import { clientKey, clientsKey } from "./useClients";
import { removeFromList, replaceInList, restoreSnapshots, snapshotQueries } from "@/lib/optimistic";

function invalidate(qc, id) {
  qc.invalidateQueries({ queryKey: clientsKey });
  qc.invalidateQueries({ queryKey: ["dashboard"] });
  if (id) qc.invalidateQueries({ queryKey: clientKey(id) });
}

export function useCreateClient() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload) => clientsApi.create(payload),
    onSuccess: () => invalidate(qc),
  });
}

// ClientFormModal catches and shows inline errors, so updates stay silent.
export function useUpdateClient() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }) => clientsApi.update(id, payload),
    onMutate: async ({ id, payload }) => {
      const prev = await snapshotQueries(qc, [clientsKey, clientKey(id)]);
      replaceInList(qc, clientsKey, (c) => c.id === id, payload);
      qc.setQueryData(clientKey(id), (old) => (old ? { ...old, client: { ...old.client, ...payload } } : old));
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
    },
    onSettled: (_c, _e, { id }) => invalidate(qc, id),
  });
}

// ClientDetail deletes without a catch, so the hook owns the toast.
export function useDeleteClient() {
  const qc = useQueryClient();
  const { t } = useLang();
  return useMutation({
    mutationFn: (id) => clientsApi.remove(id),
    onMutate: async (id) => {
      const prev = await snapshotQueries(qc, [clientsKey]);
      removeFromList(qc, clientsKey, (c) => c.id === id);
      return { prev };
    },
    onError: (_err, id, ctx) => {
      if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
      toast.error(t("clients.deleteFailed"), { id: `client-delete-${id}` });
    },
    onSettled: () => invalidate(qc),
  });
}

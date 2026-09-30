import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { itemsApi } from "@/api/items";
import { useLang } from "@/context/LangContext";
import { itemsKey } from "./useItems";
import { patchList, removeFromList, replaceInList, restoreSnapshots, snapshotQueries } from "@/lib/optimistic";

function optimisticId() {
  if (typeof crypto !== "undefined" && crypto.randomUUID) return `optimistic-${crypto.randomUUID()}`;
  return `optimistic-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function useItemMutations() {
  const qc = useQueryClient();
  const { t } = useLang();
  return {
    // ItemModal catches inline, so create/update resolve silently. Create inserts
    // a temp row (flagged), swapped for the server entity on success.
    create: useMutation({
      mutationFn: itemsApi.create,
      onMutate: async (payload) => {
        const prev = await snapshotQueries(qc, [itemsKey]);
        const temp = { ...payload, id: optimisticId(), __optimistic: true };
        patchList(qc, itemsKey, (list) => [...list, temp]);
        return { prev, tempId: temp.id };
      },
      onSuccess: (item, _payload, ctx) => {
        if (!item?.id) return;
        replaceInList(qc, itemsKey, (x) => x.id === ctx.tempId, { ...item, __optimistic: undefined });
      },
      onError: (_err, _payload, ctx) => {
        if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
      },
      onSettled: () => qc.invalidateQueries({ queryKey: itemsKey }),
    }),
    update: useMutation({
      mutationFn: ({ id, payload }) => itemsApi.update(id, payload),
      onMutate: async ({ id, payload }) => {
        const prev = await snapshotQueries(qc, [itemsKey]);
        replaceInList(qc, itemsKey, (x) => x.id === id, payload);
        return { prev };
      },
      onError: (_err, _vars, ctx) => {
        if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
      },
      onSettled: () => qc.invalidateQueries({ queryKey: itemsKey }),
    }),
    // Items list deletes without a catch, so the hook owns the toast.
    remove: useMutation({
      mutationFn: itemsApi.remove,
      onMutate: async (id) => {
        const prev = await snapshotQueries(qc, [itemsKey]);
        removeFromList(qc, itemsKey, (x) => x.id === id);
        return { prev };
      },
      onError: (_err, id, ctx) => {
        if (ctx?.prev) restoreSnapshots(qc, ctx.prev);
        toast.error(t("items.deleteFailed"), { id: `item-delete-${id}` });
      },
      onSettled: () => qc.invalidateQueries({ queryKey: itemsKey }),
    }),
  };
}

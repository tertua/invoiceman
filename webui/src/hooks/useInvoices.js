import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { invoicesApi } from "@/api/invoices";

export const invoicesKey = (params) => ["invoices", params || {}];
export const invoiceKey = (id) => ["invoice", id];

export function useInvoices(params) {
  return useQuery({
    queryKey: invoicesKey(params),
    queryFn: () => invoicesApi.list(params),
    placeholderData: keepPreviousData,
  });
}

export function useInvoice(id) {
  return useQuery({
    queryKey: invoiceKey(id),
    queryFn: () => invoicesApi.get(id),
    enabled: !!id,
  });
}

// Mutation hooks live in useInvoiceMutations.js (file-size split); re-exported
// here so callers keep importing from "@/hooks/useInvoices".
export {
  useCreateInvoice,
  useUpdateInvoice,
  useSetInvoiceStatus,
  useDeleteInvoice,
} from "./useInvoiceMutations";

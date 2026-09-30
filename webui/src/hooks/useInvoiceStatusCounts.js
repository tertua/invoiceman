import { useQuery } from "@tanstack/react-query";
import { invoiceStatusCountsApi } from "@/api/invoiceStatusCounts";

// The "invoices" prefix is what useInvoices' invalidateAll targets, so every
// invoice mutation already refreshes these counts (see hooks/useInvoices.js).
export function useInvoiceStatusCounts() {
  return useQuery({
    queryKey: ["invoices", "status-counts"],
    queryFn: invoiceStatusCountsApi.get,
    staleTime: 30_000,
  });
}

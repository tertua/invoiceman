import { useQuery } from "@tanstack/react-query";
import { reportsApi } from "@/api/reports";

export const reportsKey = ["reports"];

export function useReports(currency) {
  return useQuery({
    queryKey: [...reportsKey, currency || ""],
    queryFn: () => reportsApi.get(currency ? { currency } : {}),
  });
}

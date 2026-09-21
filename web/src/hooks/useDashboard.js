import { useQuery } from "@tanstack/react-query";
import { dashboardApi } from "@/api/dashboard";

export const dashboardKey = ["dashboard"];

export function useDashboard(currency) {
  return useQuery({
    queryKey: [...dashboardKey, currency || ""],
    queryFn: () => dashboardApi.get(currency ? { currency } : {}),
  });
}

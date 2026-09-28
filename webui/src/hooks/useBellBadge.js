import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { dashboardKey, useDashboard } from "./useDashboard";

// Bell badge data with a live refresh: the server pushes one SSE frame per
// aggregate change (payment recorded/voided, invoice created/settled) and
// this hook invalidates the dashboard query, so the overdue dot updates
// without navigation. Same return shape as useDashboard (drop-in swap).
// EventSource needs no headers: the session cookie rides along same-origin
// (via the Vite /api proxy in dev) and reconnects on its own.
export function useBellBadge(currency) {
  const qc = useQueryClient();
  const query = useDashboard(currency);
  useEffect(() => {
    const src = new EventSource("/api/v1/events");
    const refresh = () => qc.invalidateQueries({ queryKey: dashboardKey });
    for (const type of ["invoice.created", "invoice.status_updated", "payment.created", "payment.voided"]) {
      src.addEventListener(type, refresh);
    }
    return () => src.close();
  }, [qc]);
  return query;
}

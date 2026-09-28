import { settingsApi } from "@/api/settings";
import { configApi } from "@/api/config";
import { dashboardApi } from "@/api/dashboard";
import { reportsApi } from "@/api/reports";

// Runs inside the login button's minimum-delay window so the wait does
// real work: prune orphaned anon AI caches, then warm the queries the
// dashboard mounts with. Failures are swallowed — the dashboard refetches
// on mount anyway.
export async function warmSessionAfterLogin(queryClient) {
  pruneAnonAiCaches();
  try {
    const settings = await queryClient.fetchQuery({
      queryKey: ["settings"],
      queryFn: () => settingsApi.get(),
    });
    const currency = settings?.currency || "IDR";
    await Promise.allSettled([
      queryClient.prefetchQuery({
        queryKey: ["app-config"],
        queryFn: () => configApi.get(),
        staleTime: Infinity,
      }),
      queryClient.prefetchQuery({
        queryKey: ["dashboard", currency],
        queryFn: () => dashboardApi.get({ currency }),
      }),
      queryClient.prefetchQuery({
        queryKey: ["reports", currency],
        queryFn: () => reportsApi.get({ currency }),
      }),
    ]);
  } catch {
    /* unauthenticated or offline — pages fetch on mount */
  }
}

function pruneAnonAiCaches() {
  try {
    const doomed = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (
        k?.startsWith("invoiceman:ai-summary:anon") ||
        k?.startsWith("invoiceman:ai-reminder:anon")
      ) {
        doomed.push(k);
      }
    }
    doomed.forEach((k) => localStorage.removeItem(k));
  } catch {
    /* private mode — nothing to prune */
  }
}

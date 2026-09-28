import { QueryClient } from "@tanstack/react-query";

// Shared by the product entry (main.jsx) and the admin console entry (admin.jsx) so retry/stale semantics cannot drift.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => error?.status !== 401 && failureCount < 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

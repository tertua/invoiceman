import { Suspense, useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@/context/ThemeContext";
import { LangProvider, useLang } from "@/context/LangContext";
import { AuthProvider } from "@/context/AuthContext";
import { UIProvider } from "@/context/UIContext";
import { useAppName } from "@/hooks/useConfig";
import { router } from "@/routes";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => error?.status !== 401 && failureCount < 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

function PageLoading() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--bg)] text-[var(--ink-muted)] text-sm">
      Loading...
    </div>
  );
}

function DocumentTitle() {
  const { t } = useLang();
  const appName = useAppName();
  useEffect(() => {
    document.title = `${appName} — ${t("app.tagline")}`;
  }, [appName, t]);
  return null;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <LangProvider>
          <DocumentTitle />
          <UIProvider>
            <AuthProvider>
              <Suspense fallback={<PageLoading />}>
                <RouterProvider router={router} />
              </Suspense>
            </AuthProvider>
          </UIProvider>
        </LangProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

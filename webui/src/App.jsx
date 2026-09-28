import { Suspense, useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@/context/ThemeContext";
import { LangProvider, useLang } from "@/context/LangContext";
import { AuthProvider } from "@/context/AuthContext";
import { AppToaster } from "@/components/ui/AppToaster";
import { useAppName } from "@/hooks/useConfig";
import { queryClient } from "@/lib/queryClient";
import { router } from "@/routes";

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
          <AuthProvider>
            <Suspense fallback={<PageLoading />}>
              <RouterProvider router={router} />
            </Suspense>
          </AuthProvider>
          <AppToaster />
        </LangProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

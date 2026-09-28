import { StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { ThemeProvider } from "@/context/ThemeContext";
import { LangProvider } from "@/context/LangContext";
import { AuthProvider } from "@/context/AuthContext";
import { AppToaster } from "@/components/ui/AppToaster";
import { queryClient } from "@/lib/queryClient";
import { adminRouter } from "@/adminRoutes";
import "./index.css";

function PageLoading() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--bg)] text-[var(--ink-muted)] text-sm">
      Loading...
    </div>
  );
}

createRoot(document.getElementById("root")).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <LangProvider>
          <AuthProvider>
            <Suspense fallback={<PageLoading />}>
              <RouterProvider router={adminRouter} />
            </Suspense>
          </AuthProvider>
          <AppToaster />
        </LangProvider>
      </ThemeProvider>
    </QueryClientProvider>
  </StrictMode>,
);

import { lazy, useEffect } from "react";
import { Navigate, createBrowserRouter } from "react-router-dom";
import RouteError from "@/components/ui/RouteError";
import { AdminShell } from "@/components/layout/AdminShell";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";

const AdminUsers = lazy(() => import("@/pages/AdminUsers"));
const AdminGateway = lazy(() => import("@/pages/AdminGateway"));

// Leaving this entry is a full page load, so redirects use window.location — <Navigate> would be rewritten by the /admin basename.
function AdminGuard({ children }) {
  const { user, loading } = useAuth();
  const { t } = useLang();
  useEffect(() => {
    if (loading) return;
    if (!user) window.location.replace("/login");
    else if (user.role !== "admin") window.location.replace("/dashboard");
  }, [loading, user]);
  if (loading || !user || user.role !== "admin") {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--bg)] text-[var(--ink-muted)] text-sm">
        {t("common.loading")}
      </div>
    );
  }
  return children;
}

export const adminRouter = createBrowserRouter(
  [
    {
      path: "/",
      element: (
        <AdminGuard>
          <AdminShell />
        </AdminGuard>
      ),
      errorElement: <RouteError />,
      children: [
        { index: true, element: <Navigate to="users" replace /> },
        { path: "users", element: <AdminUsers /> },
        { path: "gateway", element: <AdminGateway /> },
      ],
    },
    { path: "*", element: <Navigate to="/" replace /> },
  ],
  { basename: "/admin" },
);

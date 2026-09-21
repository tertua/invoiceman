import { lazy } from "react";
import { Navigate, createBrowserRouter, useLocation } from "react-router-dom";
import { AppShell } from "@/components/layout/AppShell";
import RouteError from "@/components/ui/RouteError";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";

const Dashboard = lazy(() => import("@/pages/Dashboard"));
const Login = lazy(() => import("@/pages/Login"));
const Register = lazy(() => import("@/pages/Register"));
const ForgotPassword = lazy(() => import("@/pages/ForgotPassword"));
const ResetPassword = lazy(() => import("@/pages/ResetPassword"));
const Landing = lazy(() => import("@/pages/Landing"));
const PublicPay = lazy(() => import("@/pages/PublicPay"));
const Invoices = lazy(() => import("@/pages/Invoices"));
const InvoiceEditor = lazy(() => import("@/pages/InvoiceEditor"));
const InvoiceDetail = lazy(() => import("@/pages/InvoiceDetail"));
const Clients = lazy(() => import("@/pages/Clients"));
const ClientDetail = lazy(() => import("@/pages/ClientDetail"));
const Expenses = lazy(() => import("@/pages/Expenses"));
const Payments = lazy(() => import("@/pages/Payments"));
const Items = lazy(() => import("@/pages/Items"));
const Reports = lazy(() => import("@/pages/Reports"));
const Settings = lazy(() => import("@/pages/Settings"));
const AdminUsers = lazy(() => import("@/pages/AdminUsers"));

function ProtectedShell() {
  const { user, loading } = useAuth();
  const { t } = useLang();
  const location = useLocation();
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--bg)] text-[var(--ink-muted)] text-sm">
        {t("common.loading")}
      </div>
    );
  }
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />;
  return <AppShell />;
}

function AdminRoute() {
  const { user } = useAuth();
  return user?.role === "admin" ? <AdminUsers /> : <Navigate to="/dashboard" replace />;
}

export const router = createBrowserRouter([
  { path: "/", element: <Landing />, errorElement: <RouteError /> },
  { path: "/login", element: <Login />, errorElement: <RouteError /> },
  { path: "/register", element: <Register />, errorElement: <RouteError /> },
  { path: "/forgot-password", element: <ForgotPassword />, errorElement: <RouteError /> },
  { path: "/reset-password", element: <ResetPassword />, errorElement: <RouteError /> },
  { path: "/pay/:token", element: <PublicPay />, errorElement: <RouteError /> },
  {
    path: "/",
    element: <ProtectedShell />,
    errorElement: <RouteError />,
    children: [
      { path: "dashboard", element: <Dashboard /> },
      { path: "invoices", element: <Invoices /> },
      { path: "invoices/new", element: <InvoiceEditor /> },
      { path: "invoices/:id", element: <InvoiceDetail /> },
      { path: "invoices/:id/edit", element: <InvoiceEditor /> },
      { path: "clients", element: <Clients /> },
      { path: "clients/:id", element: <ClientDetail /> },
      { path: "expenses", element: <Expenses /> },
      { path: "payments", element: <Payments /> },
      { path: "items", element: <Items /> },
      { path: "reports", element: <Reports /> },
      { path: "settings", element: <Settings /> },
      { path: "admin/users", element: <AdminRoute /> },
    ],
  },
  { path: "*", element: <Navigate to="/" replace /> },
]);

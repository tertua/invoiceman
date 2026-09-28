import { lazy } from "react";
import { Navigate } from "react-router-dom";
import RouteError from "@/components/ui/RouteError";
import { useAllowRegistration } from "@/hooks/useConfig";

const Landing = lazy(() => import("@/pages/Landing"));
const Login = lazy(() => import("@/pages/Login"));
const Register = lazy(() => import("@/pages/Register"));
const ForgotPassword = lazy(() => import("@/pages/ForgotPassword"));
const ResetPassword = lazy(() => import("@/pages/ResetPassword"));
const PublicPay = lazy(() => import("@/pages/PublicPay"));
const PaymentFinish = lazy(() => import("@/pages/PaymentFinish"));

function RegisterRoute() {
  const allowRegistration = useAllowRegistration();
  return allowRegistration ? <Register /> : <Navigate to="/login" replace />;
}

// Standalone public routes (no session shell). Split out of routes.jsx so
// that file stays within its size ratchet.
export const publicRoutes = [
  { path: "/", element: <Landing />, errorElement: <RouteError /> },
  { path: "/login", element: <Login />, errorElement: <RouteError /> },
  { path: "/register", element: <RegisterRoute />, errorElement: <RouteError /> },
  { path: "/forgot-password", element: <ForgotPassword />, errorElement: <RouteError /> },
  { path: "/reset-password", element: <ResetPassword />, errorElement: <RouteError /> },
  { path: "/pay/:token", element: <PublicPay />, errorElement: <RouteError /> },
  { path: "/payment/finish", element: <PaymentFinish />, errorElement: <RouteError /> },
];

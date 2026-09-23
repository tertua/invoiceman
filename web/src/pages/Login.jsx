import { useState } from "react";
import { useLocation, useNavigate, Link } from "react-router-dom";
import { motion } from "framer-motion";
import { ArrowRight, ArrowLeft, Loader2, Mail, Lock } from "lucide-react";
import { AuthShell, AuthField, AuthPrimaryButton, AuthErrorBanner } from "@/components/auth/AuthShell";
import { useQueryClient } from "@tanstack/react-query";
import Turnstile from "@/components/auth/Turnstile";
import { warmSessionAfterLogin } from "@/lib/sessionWarmup";
import AILogo from "@/components/layout/AILogo";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";
import { useAllowRegistration } from "@/hooks/useConfig";

export default function Login() {
  const { login, sessionExpired, setSessionExpired } = useAuth();
  const { t } = useLang();
  const nav = useNavigate();
  const location = useLocation();

  return (
    <AuthShell
      headline={
        <>
          {t("auth.headline.login1")}
          <br />
          <em style={{ fontStyle: "italic" }}>{t("auth.headline.login2")}</em>
        </>
      }
      subhead={t("auth.login.subhead")}
    >
      <LoginForm login={login} nav={nav} location={location} t={t} sessionExpired={sessionExpired} clearSessionExpired={() => setSessionExpired(false)} />
    </AuthShell>
  );
}

function LoginForm({ login, nav, location, t, sessionExpired, clearSessionExpired }) {
  const allowRegistration = useAllowRegistration();
  const [form, setForm] = useState({ email: "", password: "" });
  const [captchaToken, setCaptchaToken] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const queryClient = useQueryClient();

  async function onSubmit(e) {
    e.preventDefault();
    setErr("");
    clearSessionExpired?.();
    setLoading(true);
    try {
      const minDelay = new Promise((r) => setTimeout(r, 800));
      await Promise.all([login(form, captchaToken).then(() => warmSessionAfterLogin(queryClient)), minDelay]);
      const destination = location.state?.from;
      nav(destination ? `${destination.pathname}${destination.search}${destination.hash}` : "/dashboard", { replace: true });
    } catch (e) {
      setErr(e.message || t("auth.loginFailed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.55, ease: [0.16, 1, 0.3, 1] }}
    >
        <div className="mb-12">
          <AILogo size={48} />
        </div>

        <h1 className="font-display text-[34px] font-semibold tracking-tight text-[var(--ink)] leading-[1.05]">
          {t("auth.welcomeBack")}
        </h1>
        <p className="text-[var(--ink-muted)] mt-2 text-[15px]">
          {t("auth.loginDesc")}
        </p>

        <form onSubmit={onSubmit} className="mt-9 space-y-4">
          <AuthField
            label={t("common.email")}
            type="email"
            autoComplete="email"
            value={form.email}
            onChange={(v) => setForm((current) => ({ ...current, email: v }))}
            placeholder="you@example.com"
            icon={Mail}
          />

          <AuthField
            label={t("common.password")}
            type="password"
            autoComplete="current-password"
            value={form.password}
            onChange={(v) => setForm((current) => ({ ...current, password: v }))}
            placeholder="••••••••"
            icon={Lock}
            extra={
              <Link
                to="/forgot-password"
                className="text-xs text-[var(--accent-strong)] font-semibold hover:underline"
              >
                {t("auth.forgot")}
              </Link>
            }
          />

          <AuthErrorBanner>{err}</AuthErrorBanner>
          {sessionExpired && !err && (
            <div className="text-xs text-[var(--warning)] bg-[var(--warning)]/10 rounded-2xl px-4 py-2.5 leading-snug">
              {t("auth.sessionExpired")}
            </div>
          )}

          <Turnstile onVerify={setCaptchaToken} />

          <div className="pt-1">
            <AuthPrimaryButton type="submit" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 size={15} className="animate-spin" />
                  {t("auth.signingIn")}
                </>
              ) : (
                <>
                  {t("auth.signIn")} <ArrowRight size={15} />
                </>
              )}
            </AuthPrimaryButton>
          </div>
        </form>

        {allowRegistration && (
          <div className="text-sm text-[var(--ink-muted)] text-center mt-8">
            {t("auth.noAccount")}{" "}
            <Link
              to="/register"
              className="text-[var(--accent-strong)] font-semibold hover:underline"
            >
              {t("auth.createOne")}
            </Link>
          </div>
        )}

        <div className="text-sm text-center mt-4">
          <Link
            to="/"
            className="inline-flex items-center gap-1.5 text-[var(--ink-muted)] font-medium hover:text-[var(--accent-strong)] hover:underline"
          >
            <ArrowLeft size={14} /> {t("auth.backToHome")}
          </Link>
        </div>
    </motion.div>
  );
}

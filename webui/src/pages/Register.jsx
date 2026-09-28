import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { motion } from "framer-motion";
import { ArrowRight, Loader2, User, Mail, Lock } from "lucide-react";
import {
  AuthShell,
  AuthField,
  AuthPrimaryButton,
  AuthErrorBanner,
} from "@/components/auth/AuthShell";
import AILogo from "@/components/layout/AILogo";
import Turnstile from "@/components/auth/Turnstile";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";

export default function Register() {
  const { register } = useAuth();
  const { t } = useLang();
  const nav = useNavigate();
  const [form, setForm] = useState({ name: "", email: "", password: "" });
  const [captchaToken, setCaptchaToken] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e) {
    e.preventDefault();
    setErr("");
    setLoading(true);
    try {
      await register(form, captchaToken);
      nav("/dashboard");
    } catch (e) {
      setErr(e.message || t("auth.registrationFailed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={
        <>
          {t("auth.headline.register1")}
          <br />
          <em style={{ fontStyle: "italic" }}>{t("auth.headline.register2")}</em>
        </>
      }
      subhead={t("auth.register.subhead")}
    >
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.55, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="mb-12">
          <AILogo size={48} />
        </div>

        <h1 className="font-display text-[34px] font-semibold tracking-tight text-[var(--ink)] leading-[1.05]">
          {t("auth.getStarted")}
        </h1>
        <p className="text-[var(--ink-muted)] mt-2 text-[15px]">
          {t("auth.freeToStart")}
        </p>

        <form onSubmit={onSubmit} className="mt-9 space-y-4">
          <AuthField
            label={t("auth.fullName")}
            autoComplete="name"
            value={form.name}
            onChange={(v) => setForm({ ...form, name: v })}
            icon={User}
            placeholder={t("auth.fullName")}
          />

          <AuthField
            label={t("common.email")}
            type="email"
            autoComplete="email"
            value={form.email}
            onChange={(v) => setForm({ ...form, email: v })}
            icon={Mail}
            placeholder={t("common.email")}
          />

          <AuthField
            label={t("common.password")}
            type="password"
            autoComplete="new-password"
            value={form.password}
            onChange={(v) => setForm({ ...form, password: v })}
            placeholder={t("auth.atLeast8")}
            minLength={8}
            icon={Lock}
          />

          <AuthErrorBanner>{err}</AuthErrorBanner>

          <Turnstile onVerify={setCaptchaToken} />

          <div className="pt-1">
            <AuthPrimaryButton type="submit" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 size={15} className="animate-spin" />
                  {t("auth.creating")}
                </>
              ) : (
                <>
                  {t("auth.createAccount")} <ArrowRight size={15} />
                </>
              )}
            </AuthPrimaryButton>
          </div>
        </form>

        <div className="text-sm text-[var(--ink-muted)] text-center mt-8">
          {t("auth.haveAccount")}{" "}
          <Link
            to="/login"
            className="text-[var(--accent-strong)] font-semibold hover:underline"
          >
            {t("auth.signIn")}
          </Link>
        </div>

        <p className="text-[11px] text-[var(--ink-muted)]/80 text-center mt-6 leading-relaxed">
          {t("auth.agreeTerms")}
          <br />
          {t("auth.neverShare")}
        </p>
      </motion.div>
    </AuthShell>
  );
}

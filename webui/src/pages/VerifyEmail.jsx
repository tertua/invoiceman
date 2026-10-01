import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { motion } from "framer-motion";
import { AlertTriangle, ArrowLeft, CheckCircle2, Loader2 } from "lucide-react";
import { AuthShell } from "@/components/auth/AuthShell";
import AILogo from "@/components/layout/AILogo";
import { authVerifyApi } from "@/api/authVerify";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";
import { VerifyResend } from "@/components/auth/VerifyPending";

// VerifyEmail redeems the ?token= from a verification email and lands the user
// in a session; a bad/expired token offers a resend by email.
export default function VerifyEmail() {
  const { t } = useLang();
  const nav = useNavigate();
  const { refresh } = useAuth();
  const [params] = useSearchParams();
  const token = params.get("token") || "";
  const [state, setState] = useState("verifying");
  const ran = useRef(false);

  useEffect(() => {
    // The link is single-use; guard against React's double effect in dev.
    if (ran.current) return;
    ran.current = true;
    if (!token) {
      setState("invalid");
      return;
    }
    authVerifyApi
      .verifyEmail({ token })
      .then(async () => {
        // The verify response only carries the user; adopt the live session
        // (me) so the shell has org context, mirroring a normal login.
        await refresh();
        setState("success");
        nav("/dashboard", { replace: true });
      })
      .catch(() => setState("invalid"));
  }, [token, refresh, nav]);

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
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.55, ease: [0.16, 1, 0.3, 1] }}
      >
        <div className="mb-12">
          <AILogo size={48} />
        </div>

        {state === "verifying" && (
          <div className="flex items-center gap-3 text-[var(--ink-muted)]">
            <Loader2 size={18} className="animate-spin" />
            <span>{t("auth.verify.verifying")}</span>
          </div>
        )}

        {state === "success" && (
          <>
            <div className="mb-6 flex h-14 w-14 items-center justify-center rounded-2xl bg-[var(--accent-soft)] text-[var(--accent-strong)]">
              <CheckCircle2 size={26} />
            </div>
            <h1 className="font-display text-[30px] font-semibold tracking-tight text-[var(--ink)] leading-[1.1]">
              {t("auth.verify.success")}
            </h1>
          </>
        )}

        {state === "invalid" && (
          <>
            <div className="mb-6 flex h-14 w-14 items-center justify-center rounded-2xl bg-[var(--danger)]/10 text-[var(--danger)]">
              <AlertTriangle size={26} />
            </div>
            <h1 className="font-display text-[30px] font-semibold tracking-tight text-[var(--ink)] leading-[1.1]">
              {t("auth.verify.invalidTitle")}
            </h1>
            <p className="text-[var(--ink-muted)] mt-3 text-[15px] leading-relaxed">
              {t("auth.verify.invalidBody")}
            </p>

            <div className="mt-8">
              <VerifyResend />
            </div>

            <div className="text-sm text-center mt-6">
              <Link
                to="/login"
                className="inline-flex items-center gap-1.5 text-[var(--ink-muted)] font-medium hover:text-[var(--accent-strong)] hover:underline"
              >
                <ArrowLeft size={14} /> {t("auth.verify.backToLogin")}
              </Link>
            </div>
          </>
        )}
      </motion.div>
    </AuthShell>
  );
}

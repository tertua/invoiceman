import { useState } from "react";
import { motion } from "framer-motion";
import { MailCheck, Loader2, ArrowLeft } from "lucide-react";
import { Link } from "react-router-dom";
import { toast } from "sonner";

import { AuthField, AuthPrimaryButton } from "@/components/auth/AuthShell";
import { authVerifyApi } from "@/api/authVerify";
import { useLang } from "@/context/LangContext";

// VerifyPending shows the "check your email" state after a register that
// created a pending account, with a resend affordance (always generic 202).
export function VerifyPending({ email }) {
  const { t } = useLang();
  const [sending, setSending] = useState(false);

  async function onResend() {
    setSending(true);
    try {
      await authVerifyApi.resendVerification({ email });
      toast.success(t("auth.verify.resendSent"));
    } catch {
      // Resend is enumeration-safe: a network error is the only real failure.
      toast.error(t("auth.verify.resendSent"));
    } finally {
      setSending(false);
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.55, ease: [0.16, 1, 0.3, 1] }}
    >
      <div className="mb-10 flex h-14 w-14 items-center justify-center rounded-2xl bg-[var(--accent-soft)] text-[var(--accent-strong)]">
        <MailCheck size={26} />
      </div>

      <h1 className="font-display text-[30px] font-semibold tracking-tight text-[var(--ink)] leading-[1.1]">
        {t("auth.verify.pendingTitle")}
      </h1>
      <p className="text-[var(--ink-muted)] mt-3 text-[15px] leading-relaxed">
        {t("auth.verify.pendingBody", { email })}
      </p>

      <div className="mt-8">
        <AuthPrimaryButton type="button" disabled={sending} onClick={onResend}>
          {sending ? (
            <>
              <Loader2 size={15} className="animate-spin" />
              {t("auth.verify.resending")}
            </>
          ) : (
            t("auth.verify.resend")
          )}
        </AuthPrimaryButton>
      </div>

      <div className="text-sm text-center mt-6">
        <Link
          to="/login"
          className="inline-flex items-center gap-1.5 text-[var(--ink-muted)] font-medium hover:text-[var(--accent-strong)] hover:underline"
        >
          <ArrowLeft size={14} /> {t("auth.verify.backToLogin")}
        </Link>
      </div>
    </motion.div>
  );
}

// VerifyResend asks for the email and re-sends the verification link; used on
// the /verify-email error state where the address is not known.
export function VerifyResend() {
  const { t } = useLang();
  const [email, setEmail] = useState("");
  const [sending, setSending] = useState(false);

  async function onSubmit(e) {
    e.preventDefault();
    setSending(true);
    try {
      await authVerifyApi.resendVerification({ email });
      toast.success(t("auth.verify.resendSent"));
    } catch {
      toast.error(t("auth.verify.resendSent"));
    } finally {
      setSending(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <AuthField
        label={t("common.email")}
        type="email"
        autoComplete="email"
        value={email}
        onChange={setEmail}
        placeholder={t("common.email")}
      />
      <AuthPrimaryButton type="submit" disabled={sending || !email}>
        {sending ? (
          <>
            <Loader2 size={15} className="animate-spin" />
            {t("auth.verify.resending")}
          </>
        ) : (
          t("auth.verify.resend")
        )}
      </AuthPrimaryButton>
    </form>
  );
}

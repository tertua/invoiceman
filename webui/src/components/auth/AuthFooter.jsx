import { Link } from "react-router-dom";
import { useLang } from "@/context/LangContext";

// Shared auth-page footer: the "have an account → sign in" link plus the terms note, extracted so Register stays within its ratchet.
export function AuthFooter() {
  const { t } = useLang();
  return (
    <>
      <div className="text-sm text-[var(--ink-muted)] text-center mt-8">
        {t("auth.haveAccount")}{" "}
        <Link to="/login" className="text-[var(--accent-strong)] font-semibold hover:underline">
          {t("auth.signIn")}
        </Link>
      </div>

      <p className="text-[11px] text-[var(--ink-muted)]/80 text-center mt-6 leading-relaxed">
        {t("auth.agreeTerms")}
        <br />
        {t("auth.neverShare")}
      </p>
    </>
  );
}

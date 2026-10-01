import { AuthShell } from "@/components/auth/AuthShell";
import { RegisterForm } from "@/components/auth/RegisterForm";
import { useLang } from "@/context/LangContext";
import { useInviteToken } from "@/hooks/useInviteToken";

export default function Register() {
  const { t } = useLang();
  const inviteToken = useInviteToken();

  return (
    <AuthShell
      headline={
        <>
          {t("auth.headline.register1")}
          <br />
          <em style={{ fontStyle: "italic" }}>{t("auth.headline.register2")}</em>
        </>
      }
      subhead={inviteToken ? t("auth.invite.subhead") : t("auth.register.subhead")}
    >
      <RegisterForm />
    </AuthShell>
  );
}

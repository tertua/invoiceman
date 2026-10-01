import { useLocation, useNavigate } from "react-router-dom";
import { AuthShell } from "@/components/auth/AuthShell";
import { LoginForm } from "@/components/auth/LoginForm";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";

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
      <LoginForm
        login={login}
        nav={nav}
        location={location}
        sessionExpired={sessionExpired}
        clearSessionExpired={() => setSessionExpired(false)}
      />
    </AuthShell>
  );
}

import { useSearchParams } from "react-router-dom";

// Reads the org invite token off the URL (?invite_token=…). Empty when absent, so callers can treat it as "no invite".
export function useInviteToken() {
  const [params] = useSearchParams();
  return params.get("invite_token") || "";
}

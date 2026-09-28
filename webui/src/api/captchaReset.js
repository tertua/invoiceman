// Cloudflare Turnstile tokens are single-use, so a failed auth attempt burns
// the token. The HTTP layer broadcasts a reset and the mounted widget mints a
// fresh one before the user retries.
export const TURNSTILE_RESET_EVENT = "tupay:turnstile-reset";

const CAPTCHA_AUTH_ENDPOINTS = ["/auth/login", "/auth/register", "/auth/forgot-password", "/auth/reset-password"];

export function shouldResetCaptcha(url) {
  return CAPTCHA_AUTH_ENDPOINTS.some((path) => url?.includes(path));
}

export function broadcastCaptchaReset() {
  try {
    window.dispatchEvent(new CustomEvent(TURNSTILE_RESET_EVENT));
  } catch {
    /* non-browser / dispatch unsupported — captcha stays valid until expiry */
  }
}
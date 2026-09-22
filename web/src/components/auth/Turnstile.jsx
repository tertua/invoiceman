import { useEffect, useRef } from "react";

const SITE_KEY = import.meta.env.VITE_TURNSTILE_SITE_KEY;

let scriptPromise = null;

function loadScript() {
  if (scriptPromise) return scriptPromise;
  scriptPromise = new Promise((resolve, reject) => {
    if (window.turnstile) {
      resolve();
      return;
    }
    const script = document.createElement("script");
    script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js";
    script.async = true;
    script.defer = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("turnstile failed to load"));
    document.head.appendChild(script);
  });
  return scriptPromise;
}

// Turnstile renders the Cloudflare challenge when VITE_TURNSTILE_SITE_KEY
// is set and reports the token via onVerify. Without a site key it renders
// nothing and the backend skips verification (dev default).
export default function Turnstile({ onVerify }) {
  const ref = useRef(null);

  useEffect(() => {
    if (!SITE_KEY || !ref.current) return undefined;
    let widgetId;
    let cancelled = false;
    loadScript().then(
      () => {
        if (cancelled || !ref.current) return;
        widgetId = window.turnstile.render(ref.current, {
          sitekey: SITE_KEY,
          callback: (token) => onVerify?.(token),
          "expired-callback": () => onVerify?.(""),
          "error-callback": () => onVerify?.(""),
        });
      },
      () => onVerify?.("")
    );
    return () => {
      cancelled = true;
      try {
        if (widgetId !== undefined) window.turnstile?.remove(widgetId);
      } catch {
        /* ignore teardown races */
      }
    };
  }, [onVerify]);

  if (!SITE_KEY) return null;
  return <div ref={ref} className="mt-1" />;
}

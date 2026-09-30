import { useCallback, useEffect, useRef, useState } from "react";
import { createPublicTransaction, newIntentKey } from "@/api/publicIntent";
import { openCheckout } from "@/lib/payerCheckout";
import { t } from "@/lib/i18n";
import CryptoWidget from "./CryptoWidget";
import QrisWidget from "./QrisWidget";
import MethodPicker from "./MethodPicker";

// PayPanel owns the payer action area: it swaps in the crypto or QRIS widget
// for those methods and keeps the MethodPicker (and its error line) visible
// otherwise. PublicPay stays a pure composer — it never talks to a gateway.
//
// Dev path — pay action panel (scaffold)
//   [done]  pick -> crypto widget / qris widget / Snap checkout, shared error
//   [next]  keep MethodPicker visible under a widget so the payer can switch
//   [later] per-method tabs instead of a mount swap
// Seam: props {token, methods, lang, gateway, onRefresh} — the panel can be
// replaced without touching PublicPay.jsx.
// End dev path
const SWAP_DELAY_MS = 300;

export default function PayPanel({ token, methods, lang, gateway, onRefresh, onStepChange }) {
  const [pending, setPending] = useState("");
  const [error, setError] = useState("");
  const [crypto, setCrypto] = useState(false);
  const [qris, setQris] = useState(false);
  const swapTimer = useRef(null);
  const keysRef = useRef({});

  useEffect(() => () => clearTimeout(swapTimer.current), []);

  // Report the panel phase up so the page's step indicator mirrors it; the
  // panel owns the swap state, the page only renders.
  useEffect(() => {
    onStepChange?.(crypto || qris ? "pay" : pending ? "pay" : "choose");
  }, [crypto, qris, pending, onStepChange]);

  const finish = useCallback(() => {
    setCrypto(false);
    setQris(false);
    setPending("");
    setError("");
    onStepChange?.("done");
    onRefresh();
  }, [onRefresh, onStepChange]);

  const fail = useCallback((message) => {
    setCrypto(false);
    setQris(false);
    setPending("");
    setError(message);
    onStepChange?.("choose");
  }, [onStepChange]);
  const back = useCallback(() => {
    setCrypto(false);
    setQris(false);
    setPending("");
    onStepChange?.("choose");
  }, [onStepChange]);

  async function pay(method) {
    if (pending) return;
    setError("");
    if (method === "crypto" || method === "qris") {
      setPending(method);
      swapTimer.current = setTimeout(() => {
        if (method === "crypto") setCrypto(true);
        else setQris(true);
      }, SWAP_DELAY_MS);
      return;
    }
    setPending(method);
    if (!keysRef.current[method]) keysRef.current[method] = newIntentKey(0);
    try {
      const res = await createPublicTransaction(token, method, undefined, keysRef.current[method]);
      // The panel dispatches on the backend-declared checkout, never on a
      // provider name. A Snap config pays in-page; anything else (hosted
      // checkout, or a response that carries no browser token) redirects
      // through the same openCheckout contract.
      const config = { ...gateway, ...res, checkout: gateway?.checkout };
      const mode = await openCheckout({
        token: res.snap_token,
        config,
        onClose: () => setPending(""),
        onError: () => setPending(""),
        onSuccess: () => onRefresh(),
      });
      if (mode !== "snap") setPending("");
    } catch (e) {
      setError(e.message || t(lang, "public.notPayable"));
      setPending("");
    }
  }

  if (crypto) {
    return <CryptoWidget key={token} token={token} lang={lang} onError={fail} onBack={back} />;
  }
  if (qris) {
    const chosen = methods.find((m) => m.id === "qris");
    return (
      <QrisWidget
        key={token}
        token={token}
        lang={lang}
        amount={chosen?.amount}
        currency={chosen?.currency}
        gateway={gateway}
        onError={fail}
        onPaid={finish}
        onBack={back}
      />
    );
  }
  return <MethodPicker methods={methods} lang={lang} onPick={pay} pending={pending} error={error} />;
}
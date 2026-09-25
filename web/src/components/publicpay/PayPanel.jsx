import { useCallback, useState } from "react";
import { publicPayApi } from "@/api/publicPay";
import { loadMidtransSnap } from "@/lib/midtrans";
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
export default function PayPanel({ token, methods, lang, gateway, onRefresh }) {
  const [pending, setPending] = useState("");
  const [error, setError] = useState("");
  const [crypto, setCrypto] = useState(false);
  const [qris, setQris] = useState(false);

  const finish = useCallback(() => {
    setCrypto(false);
    setQris(false);
    setPending("");
    setError("");
    onRefresh();
  }, [onRefresh]);

  const fail = useCallback((message) => {
    setCrypto(false);
    setQris(false);
    setPending("");
    setError(message);
  }, []);

  async function pay(method) {
    setPending(method);
    setError("");
    try {
      if (method === "crypto") {
        setCrypto(true);
        setPending("");
        return;
      }
      if (method === "qris") {
        setQris(true);
        setPending("");
        return;
      }
      const res = await publicPayApi.createTransaction(token, method);
      if (res.snap_token && gateway?.client_key) {
        const snap = await loadMidtransSnap(gateway.is_production);
        snap.pay(res.snap_token, {
          onClose: () => setPending(""),
          onError: () => setPending(""),
          onSuccess: () => onRefresh(),
        });
        return;
      }
      const hosted = res.redirect_url || res.payment_url;
      if (hosted) {
        window.location.href = hosted;
        return;
      }
      setPending("");
    } catch (e) {
      setError(e.message || t(lang, "public.notPayable"));
      setPending("");
    }
  }

  if (crypto) {
    return <CryptoWidget key={token} token={token} lang={lang} onError={fail} onPaid={finish} />;
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
      />
    );
  }
  return <MethodPicker methods={methods} lang={lang} onPick={pay} pending={pending} error={error} />;
}
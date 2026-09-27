import { useEffect, useState } from "react";
import { publicPayApi } from "@/api/publicPay";

// Preflight the crypto minimum for the asset the payer picked. The page's
// own method list is fetched without a pay currency (no asset chosen yet, so
// crypto is never hidden by some default asset's minimum); once an asset is
// picked this re-reads it with ?pay_currency= and the backend applies that
// asset's own live minimum. Fail-open — on any fetch error intent creation
// stays the authoritative check.
// Seam: (token, asset, active) — `active` only runs the request while the
// payer is still choosing, never while a payment is being created.
export function useCryptoPayable(token, asset, active) {
  const [state, setState] = useState({ checking: false, payable: true });

  useEffect(() => {
    if (!active || !asset) {
      setState({ checking: false, payable: true });
      return undefined;
    }
    let cancelled = false;
    setState({ checking: true, payable: true });
    publicPayApi
      .get(token, asset)
      .then((res) => {
        if (cancelled) return;
        const offered = (res?.methods || []).some((m) => m.id === "crypto");
        setState({ checking: false, payable: offered });
      })
      .catch(() => {
        if (!cancelled) setState({ checking: false, payable: true });
      });
    return () => {
      cancelled = true;
    };
  }, [token, asset, active]);

  return state;
}

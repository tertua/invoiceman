import { useCallback, useRef } from "react";
import { createPublicTransaction, newIntentKey } from "@/api/publicIntent";

export function useCryptoIntent() {
  const keysRef = useRef({});
  const createIntent = useCallback((token, payCurrency) => {
    const key = `${payCurrency}`;
    if (!keysRef.current[key]) keysRef.current[key] = newIntentKey(0);
    return createPublicTransaction(token, "crypto", { pay_currency: payCurrency }, keysRef.current[key]);
  }, []);
  return { createIntent };
}

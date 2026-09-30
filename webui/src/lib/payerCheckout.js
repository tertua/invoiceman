// Provider-neutral browser checkout dispatcher. This file names no provider
// *branch*: the backend tells the page which checkout to open through the
// public payload's `gateway` block (GET /gateway/config?gateway=<name>, i.e. a
// provider's PayerConfig()). loadSnap is the one concrete browser SDK asset
// today and is reached only when the backend declares a snap checkout, so
// adding a provider with its own SDK means adding a loader + one dispatch arm,
// not a provider name in a component.

let snapPromise;

// loadSnap injects the Snap widget script and resolves with window.snap. It is
// the optional provider asset: it is only reached when the backend declares a
// snap checkout (see usesSnap), so no component branches on a provider name.
export function loadSnap(isProduction = false) {
  if (window.snap) return Promise.resolve(window.snap);
  if (snapPromise) return snapPromise;
  snapPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = `https://${isProduction ? "app" : "app.sandbox"}.midtrans.com/snap/snap.js`;
    script.async = true;
    script.onload = () => window.snap ? resolve(window.snap) : reject(new Error("Snap checkout failed to load"));
    script.onerror = () => reject(new Error("Snap checkout failed to load"));
    document.head.appendChild(script);
  }).catch((e) => { snapPromise = undefined; throw e; });
  return snapPromise;
}

// usesSnap decides whether `token` must be opened with the Snap SDK, from the
// config alone — no provider name is ever consulted. Two rules, in order:
//
//  1. config.checkout is the provider-declared mode (e.g. "snap" from
//     platform/midtrans). Any other value is a checkout this page does not
//     embed, so the caller redirects instead.
//  2. A config with no `checkout` key at all is a legacy payer config (older
//     cached bundle or a client reading the direct API): there the presence of
//     a public client_key is the provider's declaration that its token is a
//     widget token, exactly what the panel used to test inline.
//
// Rule 2 keeps a real redirect fallback for hosted methods when the
// declaration is missing; a config that explicitly says "not snap" never
// loads the SDK.
export function usesSnap({ token, config } = {}) {
  if (!token || !config) return false;
  if (config.checkout !== undefined && config.checkout !== null) return config.checkout === "snap";
  return !!config.client_key;
}

// openCheckout opens whatever the backend said the token needs:
//   - Snap widget token  -> load the SDK and pay in-page (onSuccess/onClose/onError)
//   - hosted checkout    -> navigate to redirect_url || payment_url
//   - neither            -> report through onError; the caller owns the UI state
// It only touches window.location on the hosted path, so a Snap checkout never
// triggers a redirect. `loader` is injectable for tests.
export async function openCheckout({ token, config, onSuccess, onClose, onError, loader = loadSnap } = {}) {
  if (usesSnap({ token, config })) {
    const snap = await loader(config?.is_production);
    snap.pay(token, {
      onClose,
      onError,
      onSuccess,
    });
    return "snap";
  }
  const hosted = config?.redirect_url || config?.payment_url;
  if (hosted) {
    window.location.href = hosted;
    return "redirect";
  }
  onError?.(new Error("no checkout available"));
  return "error";
}

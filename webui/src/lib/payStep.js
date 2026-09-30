// Resolves the public-pay step indicator from page + panel state. Pure so the
// three-state machine is unit-testable without a DOM.
//
//   isPaid   — invoice already settled (page state)        -> "done"
//   panelStep— PayPanel's own phase ("choose" | "pay" | "done")
//   canPay   — invoice is payable at all
//
// Once paid the invoice always wins, regardless of stale panel state.
export function resolvePayStep({ isPaid, canPay, panelStep }) {
  if (isPaid) return "done";
  if (!canPay) return "choose";
  if (panelStep === "done") return "done";
  if (panelStep === "pay") return "pay";
  return "choose";
}

// EN ui namespace — split out of i18n.en.js (file-size ratchet).
// Holds the UI-polish keys: dashboard aging/delta labels, invoice status
// counts, and the public-pay step indicator + expiry copy.
export const uiEn = {
    /* ============================ ui polish ============================ */
    "dash.aging30": "1–30 days",
    "dash.aging60": "31–60 days",
    "dash.aging90": "61–90 days",
    "dash.aging90plus": "90+ days",
    "dash.agingCurrent": "Not due",
    "dash.deltaUp": "vs last month",
    "dash.receivablesAging": "Receivables aging",
    "dash.receivablesAgingDesc": "Outstanding balance by days overdue",
    "invoices.statusCountsFailed": "Couldn't load counts",
    /* optimistic mutation toasts — only hooks whose callers have no error UI */
    "invoices.statusFailed": "Couldn't update invoice status",
    "invoices.deleteFailed": "Couldn't delete invoice",
    "payments.voidFailed": "Couldn't void payment",
    "clients.deleteFailed": "Couldn't delete client",
    "expenses.deleteFailed": "Couldn't delete expense",
    "items.deleteFailed": "Couldn't delete item",
    "public.stepChoose": "Choose method",
    "public.stepPay": "Pay",
    "public.stepDone": "Done",
    "public.expiresIn": "Expires in {time}",
    "public.invoiceExpired": "This payment window has expired",
    "public.successPaidTitle": "Payment received",
    "finish.successAnimAria": "Payment successful",
    /* invite-only registration */
    "auth.invite.subhead": "You've been invited — create your account to join the team.",
};

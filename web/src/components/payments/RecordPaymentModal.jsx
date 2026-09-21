import { useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { X, Loader2, Copy, Check, ExternalLink, Send, Link2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useInvoices } from "@/hooks/useInvoices";
import { usePaymentMutations } from "@/hooks/usePayments";
import { paymentsApi } from "@/api/payments";
import { gatewayApi } from "@/api/gateway";
import { useLang } from "@/context/LangContext";
import { formatMoney, todayDateInput } from "@/lib/utils";
import { loadMidtransSnap } from "@/lib/midtrans";

export const PAYMENT_METHODS = ["Cash", "Bank transfer", "Online"];

function Field({ label, children }) {
  return (
    <label className="block">
      <span className="block text-xs font-medium text-[var(--ink-muted)] mb-1.5">{label}</span>
      {children}
    </label>
  );
}

const selectClass =
  "h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15";

export function RecordPaymentModal({ open, onClose, invoiceId, invoiceNumber, amount, defaultEmail }) {
  const { t } = useLang();
  const { data: invoices } = useInvoices();
  const { create } = usePaymentMutations();
  const [form, setForm] = useState({
    invoiceId: invoiceId || "",
    amount: "",
    method: "Cash",
    paid_on: "",
    notes: "",
  });
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  // Online link flow state
  const [link, setLink] = useState(null); // { url, token }
  const [sendEmail, setSendEmail] = useState("");
  const [emailState, setEmailState] = useState(""); // "" | "sending" | "sent" | "error"
  const [emailErr, setEmailErr] = useState("");
  const [copied, setCopied] = useState(false);
  const [onlinePaying, setOnlinePaying] = useState(false);

  const fixed = !!invoiceId;

  const options = useMemo(() => {
    const all = invoices || [];
    return all.filter((i) => i.effective_status !== "paid").concat(all.filter((i) => i.effective_status === "paid"));
  }, [invoices]);

  const selectedInvoice = useMemo(
    () => (invoices || []).find((i) => i.id === form.invoiceId),
    [invoices, form.invoiceId]
  );

  // Draft invoices cannot be paid online — hide the Online method.
  const methodOptions = useMemo(() => {
    if (selectedInvoice?.effective_status === "draft") return PAYMENT_METHODS.filter((m) => m !== "Online");
    return PAYMENT_METHODS;
  }, [selectedInvoice]);

  useEffect(() => {
    if (open) {
      setForm({
        invoiceId: invoiceId || "",
        amount: amount != null ? String(amount) : "",
        method: "Cash",
        paid_on: todayDateInput(),
        notes: "",
      });
      setLink(null);
      setErr("");
      setEmailState("");
      setEmailErr("");
      setSendEmail(defaultEmail || "");
      setCopied(false);
    }
  }, [open, invoiceId, amount, defaultEmail]);

  useEffect(() => {
    if (!open) return;
    function onKey(e) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  function pickInvoice(id) {
    const inv = (invoices || []).find((i) => i.id === id);
    setForm((f) => ({ ...f, invoiceId: id, amount: inv ? inv.total : f.amount }));
    if (inv?.effective_status === "draft") {
      setForm((f) => (f.method === "Online" ? { ...f, method: "Cash" } : f));
    }
  }

  async function onSubmit(e) {
    e.preventDefault();
    if (!form.invoiceId) return setErr(t("payments.selectInvoice"));
    if (!(Number(form.amount) > 0)) return setErr(t("payments.validAmount"));

    if (form.method === "Online") {
      if (selectedInvoice?.currency && selectedInvoice.currency !== "IDR")
        return setErr(t("payments.onlineOnlyIdr"));
      setSaving(true);
      setErr("");
      try {
        const res = await paymentsApi.createOnlineLink(form.invoiceId);
        setLink(res);
        setSendEmail(defaultEmail || selectedInvoice?.client_email || "");
      } catch (ex) {
        setErr(ex.message || t("payments.saveFailed"));
      } finally {
        setSaving(false);
      }
      return;
    }

    setSaving(true);
    setErr("");
    try {
      await create.mutateAsync({ ...form, amount: Number(form.amount) });
      onClose();
    } catch (ex) {
      setErr(ex.message || t("payments.saveFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(link.url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable */
    }
  }

  async function sendLink() {
    if (!sendEmail.trim()) return setEmailErr(t("payments.selectInvoice"));
    setEmailState("sending");
    setEmailErr("");
    try {
      await paymentsApi.sendOnlineLink(form.invoiceId, sendEmail.trim());
      setEmailState("sent");
    } catch (ex) {
      setEmailState(ex.status === 501 ? "unavailable" : "error");
      setEmailErr(ex.status === 501 ? t("payments.emailUnavailable") : ex.message || t("payments.saveFailed"));
    }
  }

  async function payInvoice() {
    setOnlinePaying(true);
    setEmailErr("");
    try {
      const config = await gatewayApi.config();
      if (!config.client_key) throw Object.assign(new Error(t("payments.gatewayUnavailable")), { status: 501 });
      const intent = await gatewayApi.createInvoiceIntent(form.invoiceId);
      const snap = await loadMidtransSnap(config.is_production);
      snap.pay(intent.snap_token, { onClose: () => setOnlinePaying(false), onError: () => setOnlinePaying(false) });
    } catch (ex) {
      setEmailErr(ex.message || t("payments.saveFailed"));
      setOnlinePaying(false);
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div className="fixed inset-0 z-50 flex items-center justify-center p-4" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
          <div className="absolute inset-0 bg-[var(--ink)]/30 backdrop-blur-sm" onClick={onClose} />
          <motion.div
            initial={{ opacity: 0, y: 12, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.98 }}
            transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
            className="relative w-full max-w-[480px] rounded-3xl bg-[var(--surface)] border border-[var(--border)] shadow-hover p-6"
          >
            {link ? (
              <>
                <div className="flex items-center justify-between mb-5">
                  <h3 className="font-display text-lg font-semibold tracking-tight">{t("payments.onlineTitle")}</h3>
                  <button type="button" onClick={onClose} className="h-8 w-8 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface-2)]"><X size={16} /></button>
                </div>
                <div className="flex items-start gap-3 mb-4">
                  <div className="h-9 w-9 shrink-0 rounded-xl bg-[var(--accent-soft)] text-[var(--accent-strong)] flex items-center justify-center">
                    <Link2 size={16} />
                  </div>
                  <p className="text-sm text-[var(--ink-muted)] leading-snug">{t("payments.onlineDesc")}</p>
                </div>
                <div className="flex items-center gap-2 rounded-2xl border border-[var(--border)] bg-[var(--surface-2)] px-4 py-3 mb-4">
                  <span className="flex-1 min-w-0 text-sm text-[var(--ink)] truncate">{link.url}</span>
                  <button type="button" onClick={copyLink} className="shrink-0 inline-flex items-center gap-1 text-[11px] font-semibold text-[var(--accent-strong)]">
                    {copied ? <Check size={12} /> : <Copy size={12} />}
                    {copied ? t("payments.onlineCopied") : t("payments.onlineCopy")}
                  </button>
                </div>
                <a href={link.url} target="_blank" rel="noreferrer" className="block">
                  <Button variant="outline" className="w-full mb-3">
                    <ExternalLink size={14} /> {t("payments.onlineOpen")}
                  </Button>
                </a>
                <Button variant="accent" className="w-full mb-3" onClick={payInvoice} disabled={onlinePaying}>
                  {onlinePaying ? <Loader2 size={14} className="animate-spin" /> : <ExternalLink size={14} />}
                  {t("payments.onlinePayNow")}
                </Button>
                <div className="flex items-center gap-2">
                  <Input value={sendEmail} onChange={(e) => setSendEmail(e.target.value)} placeholder={t("payments.onlineEmailPlaceholder")} />
                  <Button variant="accent" onClick={sendLink} disabled={emailState === "sending"}>
                    {emailState === "sending" ? <Loader2 size={14} className="animate-spin" /> : <Send size={14} />}
                    {emailState === "sent" ? t("payments.onlineSent") : t("payments.onlineSend")}
                  </Button>
                </div>
                {emailErr && <p className="text-sm text-[var(--danger)] mt-3">{emailErr}</p>}
                <div className="flex items-center justify-end gap-2 mt-6">
                  <Button type="button" variant="outline" onClick={onClose}>{t("common.done")}</Button>
                </div>
              </>
            ) : (
              <form onSubmit={onSubmit}>
                <div className="flex items-center justify-between mb-5">
                  <h3 className="font-display text-lg font-semibold tracking-tight">{t("payments.recordTitle")}</h3>
                  <button type="button" onClick={onClose} className="h-8 w-8 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface-2)]"><X size={16} /></button>
                </div>
                <div className="space-y-3">
                  {fixed ? (
                    <Field label={`${t("common.invoice")} *`}>
                      <div className="h-10 flex items-center rounded-full border border-[var(--border)] bg-[var(--surface-2)] px-4 text-sm font-semibold text-[var(--ink)]">
                        {invoiceNumber || "—"}
                      </div>
                    </Field>
                  ) : (
                    <Field label={`${t("common.invoice")} *`}>
                      <select className={selectClass} value={form.invoiceId} onChange={(e) => pickInvoice(e.target.value)}>
                        <option value="">{t("payments.selectOption")}</option>
                        {options.map((i) => (
                          <option key={i.id} value={i.id}>
                            {i.invoice_number} · {i.client_name || t("common.noClient")} · {formatMoney(i.total, i.currency)}{i.effective_status === "paid" ? ` ${t("payments.paid")}` : ""}
                          </option>
                        ))}
                      </select>
                    </Field>
                  )}
                  {form.method !== "Online" && (
                    <div className="grid grid-cols-2 gap-3">
                      <Field label={`${t("common.amount")} *`}>
                        <Input type="number" min="0" step="0.01" value={form.amount} onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))} className="tabular" placeholder="0.00" />
                      </Field>
                      <Field label={t("common.date")}>
                        <Input type="date" value={form.paid_on} onChange={(e) => setForm((f) => ({ ...f, paid_on: e.target.value }))} />
                      </Field>
                    </div>
                  )}
                  <Field label={t("common.method")}>
                    <select className={selectClass} value={form.method} onChange={(e) => setForm((f) => ({ ...f, method: e.target.value }))}>
                      {methodOptions.map((m) => <option key={m} value={m}>{m}</option>)}
                    </select>
                  </Field>
                  {form.method !== "Online" && (
                    <Field label={t("common.notes")}>
                      <Input value={form.notes} onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))} placeholder={t("payments.notesPlaceholder")} />
                    </Field>
                  )}
                </div>
                {err && <p className="text-sm text-[var(--danger)] mt-3">{err}</p>}
                <div className="flex items-center justify-end gap-2 mt-6">
                  <Button type="button" variant="outline" onClick={onClose}>{t("common.cancel")}</Button>
                  <Button type="submit" variant="accent" disabled={saving}>
                    {saving && <Loader2 size={14} className="animate-spin" />}
                    {t("payments.recordTitle")}
                  </Button>
                </div>
              </form>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

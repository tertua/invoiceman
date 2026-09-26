import { useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { X, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useInvoices } from "@/hooks/useInvoices";
import { usePaymentMutations } from "@/hooks/usePayments";
import { useLang } from "@/context/LangContext";
import { formatMoney, todayDateInput } from "@/lib/utils";
import { PAYMENT_METHODS } from "@/lib/paymentMethods";

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

const MANUAL_METHODS = PAYMENT_METHODS.filter((m) => m !== "Online");

export function RecordPaymentModal({ open, onClose, invoiceId, invoiceNumber, amount }) {
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

  const fixed = !!invoiceId;

  const options = useMemo(() => {
    const all = invoices || [];
    return all.filter((i) => i.effective_status !== "paid" && i.effective_status !== "pending").concat(all.filter((i) => i.effective_status === "paid" || i.effective_status === "pending"));
  }, [invoices]);

  useEffect(() => {
    if (open) {
      setForm({
        invoiceId: invoiceId || "",
        amount: amount != null ? String(amount) : "",
        method: "Cash",
        paid_on: todayDateInput(),
        notes: "",
      });
      setErr("");
    }
  }, [open, invoiceId, amount]);

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
  }

  async function onSubmit(e) {
    e.preventDefault();
    if (!form.invoiceId) return setErr(t("payments.selectInvoice"));
    if (!(Number(form.amount) > 0)) return setErr(t("payments.validAmount"));
    setSaving(true);
    setErr("");
    try {
      await create.mutateAsync({ ...form, amount: form.amount });
      onClose();
    } catch (ex) {
      if (ex.status !== 401) setErr(ex.message || t("payments.saveFailed"));
    } finally {
      setSaving(false);
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
                <div className="grid grid-cols-2 gap-3">
                  <Field label={`${t("common.amount")} *`}>
                    <Input type="number" min="0" step="0.01" value={form.amount} onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))} className="tabular" placeholder="0.00" />
                  </Field>
                  <Field label={t("common.date")}>
                    <Input type="date" value={form.paid_on} onChange={(e) => setForm((f) => ({ ...f, paid_on: e.target.value }))} />
                  </Field>
                </div>
                <Field label={t("common.method")}>
                  <select className={selectClass} value={form.method} onChange={(e) => setForm((f) => ({ ...f, method: e.target.value }))}>
                    {MANUAL_METHODS.map((m) => <option key={m} value={m}>{m}</option>)}
                  </select>
                </Field>
                <Field label={t("common.notes")}>
                  <Input value={form.notes} onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))} placeholder={t("payments.notesPlaceholder")} />
                </Field>
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
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

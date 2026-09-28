import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { X, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useLang } from "@/context/LangContext";
import { formatMoney } from "@/lib/utils";

export function VoidPaymentModal({ open, onClose, payment, currency, onVoid }) {
  const { t } = useLang();
  const [reason, setReason] = useState("");
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setReason("");
      setErr("");
      setSaving(false);
    }
  }, [open, payment?.id]);

  useEffect(() => {
    if (!open) return;
    function onKey(e) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  async function onSubmit(e) {
    e.preventDefault();
    if (!reason.trim()) return setErr(t("payments.voidReasonRequired"));
    setSaving(true);
    setErr("");
    try {
      await onVoid(reason.trim());
      onClose();
    } catch (ex) {
      if (ex.status !== 401) setErr(ex.message || t("payments.voidFailed"));
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
            className="relative w-full max-w-[440px] rounded-3xl bg-[var(--surface)] border border-[var(--border)] shadow-hover p-6"
          >
            <form onSubmit={onSubmit}>
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-display text-lg font-semibold tracking-tight">{t("payments.voidTitle")}</h3>
                <button type="button" onClick={onClose} className="h-8 w-8 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface-2)]"><X size={16} /></button>
              </div>
              <p className="text-sm text-[var(--ink-muted)] leading-snug mb-4">
                {t("payments.voidDesc", { amount: payment ? formatMoney(payment.amount, currency) : "" })}
              </p>
              <label className="block">
                <span className="block text-xs font-medium text-[var(--ink-muted)] mb-1.5">{t("payments.voidReason")}</span>
                <Input
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder={t("payments.voidReasonPlaceholder")}
                  maxLength={500}
                  autoFocus
                />
              </label>
              {err && <p className="text-sm text-[var(--danger)] mt-3">{err}</p>}
              <div className="flex items-center justify-end gap-2 mt-6">
                <Button type="button" variant="outline" onClick={onClose}>{t("common.cancel")}</Button>
                <Button type="submit" variant="accent" disabled={saving || !reason.trim()}>
                  {saving && <Loader2 size={14} className="animate-spin" />}
                  {t("payments.voidConfirm")}
                </Button>
              </div>
            </form>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

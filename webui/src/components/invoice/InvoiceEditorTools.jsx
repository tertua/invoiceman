import { useEffect, useRef, useState } from "react";
import { Loader2, Package, ScanLine, X } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { useItems } from "@/hooks/useItems";
import { aiApi, isAiUnavailable, isAiFailure, isAiRateLimited } from "@/api/ai";
import { useLang } from "@/context/LangContext";
import { formatMoney } from "@/lib/utils";

export function CatalogPicker({ onPick, currency }) {
  const { t } = useLang();
  const { data: items } = useItems();
  const [open, setOpen] = useState(false);
  const ref = useRef(null);

  useEffect(() => {
    if (!open) return;
    const onClick = (e) => {
      if (!ref.current?.contains(e.target)) setOpen(false);
    };
    window.addEventListener("mousedown", onClick);
    return () => window.removeEventListener("mousedown", onClick);
  }, [open]);

  if (!items?.length) return null;

  return (
    <div ref={ref} className="relative">
      <Button type="button" variant="soft" size="sm" onClick={() => setOpen((v) => !v)}>
        <Package size={13} /> {t("invEditor.fromCatalog")}
      </Button>
      {open && (
        <div className="absolute right-0 top-10 z-20 w-64 max-h-72 overflow-y-auto rounded-2xl bg-[var(--surface)] border border-[var(--border)] shadow-hover p-1.5">
          {items.map((it) => (
            <button
              key={it.id}
              type="button"
              onClick={() => {
                onPick(it);
                setOpen(false);
              }}
              className="w-full flex items-center justify-between gap-2 px-3 py-2 rounded-xl text-left hover:bg-[var(--surface-2)] transition-colors"
            >
              <span className="text-sm text-[var(--ink)] truncate">{it.name}</span>
              <span className="text-xs font-semibold tabular text-[var(--accent-strong)] shrink-0">
                {formatMoney(it.rate, currency)}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export function ReceiptScanButton({ onParsed }) {
  const { t } = useLang();
  const inputRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");
  const [unavailable, setUnavailable] = useState(false);

  async function onFile(e) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setErr("");
    setLoading(true);
    try {
      const res = await aiApi.receiptParse(file);
      onParsed(res);
    } catch (ex) {
      setUnavailable(isAiUnavailable(ex));
      if (ex.status !== 401) setErr(isAiUnavailable(ex) ? t("ai.unavailable") : isAiRateLimited(ex) ? t("ai.rateLimited") : isAiFailure(ex) ? t("ai.failed") : ex.message || t("invEditor.scanFailed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex items-center gap-2">
      {err && (
        <span className="text-[11px] text-[var(--danger)] flex items-center gap-1">
          {err}
          <button type="button" onClick={() => setErr("")}>
            <X size={11} />
          </button>
        </span>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="image/*,application/pdf"
        className="hidden"
        onChange={onFile}
      />
      <Button variant="soft" size="sm" onClick={() => inputRef.current?.click()} disabled={loading || unavailable}>
        {loading ? <Loader2 size={13} className="animate-spin" /> : <ScanLine size={13} />}
        {t("invEditor.scanReceipt")}
      </Button>
    </div>
  );
}

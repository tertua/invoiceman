import { useEffect, useState } from "react";
import { Loader2, Sparkles, Copy, Check } from "lucide-react";
import { Card, CardTitle } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { useLang } from "@/context/LangContext";
import { useAuth } from "@/context/AuthContext";
import { cn } from "@/lib/utils";
import { aiApi, isAiUnavailable, isAiFailure, isAiRateLimited } from "@/api/ai";

const TONES = [
  { key: "friendly", labelKey: "invDetail.toneFriendly" },
  { key: "firm", labelKey: "invDetail.toneFirm" },
  { key: "final", labelKey: "invDetail.toneFinal" },
];

// The last generated draft survives menu switches and reloads via
// localStorage, scoped per user, invoice, tone and language; regenerating
// overwrites it.
const REMINDER_KEY_PREFIX = "invoiceman:ai-reminder:";

function reminderKey(userId, invoiceId, tone, lang) {
  return `${REMINDER_KEY_PREFIX}${userId || "anon"}:${invoiceId}:${tone}:${lang === "id" ? "id" : "en"}`;
}

function loadCachedDraft(key) {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (typeof parsed?.draft?.subject === "string" && typeof parsed?.draft?.body === "string") {
      return parsed.draft;
    }
    return null;
  } catch {
    return null;
  }
}

export function InvoiceReminderCard({ invoiceId }) {
  const { t, lang } = useLang();
  const { user } = useAuth();
  const [tone, setTone] = useState("friendly");
  const key = reminderKey(user?.id, invoiceId, tone, lang);
  const [draft, setDraft] = useState(() => loadCachedDraft(key));
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");
  const [unavailable, setUnavailable] = useState(false);
  const [copied, setCopied] = useState(false);

  // Pick up the cache for the active user/invoice/tone/language.
  useEffect(() => {
    setDraft(loadCachedDraft(key));
  }, [key]);

  async function generate() {
    setLoading(true);
    setErr("");
    setUnavailable(false);
    try {
      const res = await aiApi.paymentReminder(invoiceId, tone);
      setDraft(res.draft);
      try {
        localStorage.setItem(key, JSON.stringify({ draft: res.draft, at: Date.now() }));
      } catch {
        /* private mode / quota — in-memory draft still shows */
      }
    } catch (e) {
      setUnavailable(isAiUnavailable(e));
      if (e.status !== 401) setErr(isAiUnavailable(e) ? t("ai.unavailable") : isAiRateLimited(e) ? t("ai.rateLimited") : isAiFailure(e) ? t("ai.failed") : e.message || t("invDetail.generateFailed"));
    } finally {
      setLoading(false);
    }
  }

  function copy() {
    const text = `Subject: ${draft.subject}\n\n${draft.body}`;
    navigator.clipboard?.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <Card padding="lg">
      <div className="flex items-center gap-2 mb-1">
        <div className="h-8 w-8 rounded-xl bg-[var(--accent-soft)] text-[var(--accent-strong)] flex items-center justify-center">
          <Sparkles size={15} />
        </div>
        <CardTitle>{t("invDetail.aiReminder")}</CardTitle>
      </div>
      <p className="text-xs text-[var(--ink-muted)] mb-3">
        {t("invDetail.reminderDesc")}
      </p>

      <div className="flex items-center gap-1 p-1 rounded-full bg-[var(--surface-2)] mb-3">
        {TONES.map((toneItem) => (
          <button type="button"
            key={toneItem.key}
            onClick={() => setTone(toneItem.key)}
            className={cn(
              "flex-1 h-7 rounded-full text-[11px] font-semibold transition-colors",
              tone === toneItem.key ? "bg-[var(--surface)] text-[var(--ink)] shadow-card" : "text-[var(--ink-muted)]"
            )}
          >
            {t(toneItem.labelKey)}
          </button>
        ))}
      </div>

      <Button variant="accent" size="sm" className="w-full" onClick={generate} disabled={loading || unavailable}>
        {loading ? <Loader2 size={13} className="animate-spin" /> : <Sparkles size={13} />}
        {draft ? t("invDetail.regenerate") : t("invDetail.generateDraft")}
      </Button>

      {err && <p className={`text-xs mt-3 ${unavailable ? "text-[var(--ink-muted)]" : "text-[var(--danger)]"}`}>{err}</p>}

      {draft && (
        <div className="mt-4 rounded-2xl border border-[var(--border)] bg-[var(--surface-2)] p-4">
          <div className="flex items-center justify-between gap-2 mb-2">
            <div className="text-xs font-semibold text-[var(--ink)] truncate">{draft.subject}</div>
            <button type="button"
              onClick={copy}
              className="shrink-0 inline-flex items-center gap-1 text-[11px] font-semibold text-[var(--accent-strong)]"
            >
              {copied ? <Check size={12} /> : <Copy size={12} />}
              {copied ? t("invDetail.copied") : t("invDetail.copy")}
            </button>
          </div>
          <p className="text-[13px] leading-relaxed text-[var(--ink)] whitespace-pre-wrap">{draft.body}</p>
        </div>
      )}
    </Card>
  );
}

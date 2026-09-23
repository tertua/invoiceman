import { useEffect, useState } from "react";
import { useNavigate, useParams, Link } from "react-router-dom";
import {
  ArrowLeft,
  Pencil,
  Trash2,
  Loader2,
  Send,
  Undo2,
  Sparkles,
  Copy,
  Check,
  Mail,
  Plus,
  Wallet,
  Link2,
  ExternalLink,
} from "lucide-react";
import { Card, CardTitle } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { StatusBadge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import { InvoicePdfDownload } from "@/components/invoice/InvoicePdfDownload";
import {
  useInvoice,
  useSetInvoiceStatus,
  useDeleteInvoice,
} from "@/hooks/useInvoices";
import { useSettings } from "@/hooks/useSettings";
import { usePaymentMutations } from "@/hooks/usePayments";
import { RecordPaymentModal } from "@/components/payments/RecordPaymentModal";
import { paymentsApi } from "@/api/payments";
import { aiApi, isAiUnavailable, isAiFailure, isAiRateLimited } from "@/api/ai";
import { useLang } from "@/context/LangContext";
import { useAuth } from "@/context/AuthContext";
import { formatMoney, formatDate, cn } from "@/lib/utils";

export default function InvoiceDetail() {
  const { id } = useParams();
  const nav = useNavigate();
  const { t, lang } = useLang();
  const { data: invoice, isLoading, error } = useInvoice(id);
  const { data: settings } = useSettings();
  const setStatus = useSetInvoiceStatus();
  const del = useDeleteInvoice();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24 text-[var(--ink-muted)]">
        <Loader2 className="animate-spin" size={20} />
      </div>
    );
  }
  if (error?.status === 401) return null;
  if (error || !invoice) {
    return <EmptyState icon={Mail} title={t("invDetail.notFound")} description={t("invDetail.deleted")} />;
  }

  const st = invoice.effective_status;
  const isPaid = invoice.status === "paid";

  async function onDelete() {
    if (!window.confirm(t("invDetail.confirmDelete", { number: invoice.invoice_number }))) return;
    await del.mutateAsync(id);
    nav("/invoices");
  }

  return (
    <div className="max-w-[1100px]">
      {/* header */}
      <div className="flex items-center justify-between gap-4 mb-6 flex-wrap">
        <div className="flex items-center gap-3">
          <button type="button"
            onClick={() => nav("/invoices")}
            className="h-9 w-9 rounded-full flex items-center justify-center border border-[var(--border)] bg-[var(--surface)] text-[var(--ink-muted)] hover:text-[var(--ink)] shadow-card"
          >
            <ArrowLeft size={16} />
          </button>
          <div>
            <div className="flex items-center gap-3">
              <h2 className="font-display text-2xl font-semibold tracking-tight">
                {invoice.invoice_number}
              </h2>
              <StatusBadge status={st} />
            </div>
            <p className="text-sm text-[var(--ink-muted)]">
              {invoice.client_name || t("common.noClient")} · {formatMoney(invoice.total, invoice.currency)}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          <InvoicePdfDownload invoice={invoice} settings={settings} lang={lang} label="PDF" />
          <Button variant="outline" onClick={() => nav(`/invoices/${id}/edit`)}>
            <Pencil size={15} /> {t("common.edit")}
          </Button>
          <Button
            variant="ghost"
            onClick={onDelete}
            className="text-[var(--danger)] hover:bg-[var(--danger)]/10"
          >
            <Trash2 size={15} />
          </Button>
        </div>
      </div>

      {/* status controls */}
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <span className="text-xs text-[var(--ink-muted)] mr-1">{t("invDetail.markAs")}</span>
        <StatusButton
          active={invoice.status === "draft"}
          onClick={() => setStatus.mutate({ id, status: "draft" })}
          icon={Undo2}
          label={t("status.draft")}
        />
        <StatusButton
          active={invoice.status === "sent"}
          onClick={() => setStatus.mutate({ id, status: "sent" })}
          icon={Send}
          label={t("status.sent")}
        />
        {setStatus.isPending && <Loader2 size={14} className="animate-spin text-[var(--ink-muted)]" />}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Invoice preview */}
        <div className="lg:col-span-2">
          <InvoicePreview invoice={invoice} settings={settings} />
        </div>

        {/* Side: payment + reminder + client */}
        <div className="space-y-5">
          <PaymentCard invoice={invoice} />
          {!isPaid && <PaymentReminderCard invoiceId={id} />}
          <ClientCard invoice={invoice} />
        </div>
      </div>
    </div>
  );
}

function StatusButton({ active, onClick, icon: Icon, label, tone }) {
  return (
    <button type="button"
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1.5 h-8 px-3 rounded-full text-xs font-semibold border transition-colors",
        active
          ? tone === "success"
            ? "bg-[var(--success)]/12 text-[var(--success)] border-transparent"
            : "bg-[var(--ink)] text-[var(--bg)] border-transparent"
          : "bg-[var(--surface)] text-[var(--ink-muted)] border-[var(--border)] hover:text-[var(--ink)]"
      )}
    >
      <Icon size={13} />
      {label}
    </button>
  );
}

function InvoicePreview({ invoice, settings }) {
  const { t } = useLang();
  const s = settings || {};
  const currency = invoice.currency;
  return (
    <Card padding="lg">
      <div className="flex items-start justify-between gap-4 pb-6 border-b border-[var(--border)]">
        <div>
          {s.logo_url ? (
            <img src={s.logo_url} alt="" className="h-12 w-12 object-contain mb-2 rounded" />
          ) : null}
          <div className="font-display text-lg font-semibold text-[var(--ink)]">
            {s.company_name || "Your Company"}
          </div>
          {s.address && <div className="text-xs text-[var(--ink-muted)] max-w-[220px]">{s.address}</div>}
          {s.email && <div className="text-xs text-[var(--ink-muted)]">{s.email}</div>}
        </div>
        <div className="text-right">
          <div className="font-display text-2xl font-bold tracking-wide text-[var(--accent-strong)]">
            {t("common.invoice").toUpperCase()}
          </div>
          <div className="text-sm text-[var(--ink-muted)] mt-1 tabular">{invoice.invoice_number}</div>
        </div>
      </div>

      <div className="flex items-start justify-between gap-4 py-6">
        <div>
          <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">
            {t("invDetail.billTo")}
          </div>
          <div className="text-sm font-semibold text-[var(--ink)]">{invoice.client_name || "—"}</div>
          {invoice.client_company && <div className="text-xs text-[var(--ink-muted)]">{invoice.client_company}</div>}
          {invoice.client_email && <div className="text-xs text-[var(--ink-muted)]">{invoice.client_email}</div>}
        </div>
        <div className="text-right text-sm space-y-1">
          <MetaLine label={t("invDetail.issued")} value={formatDate(invoice.issue_date)} />
          <MetaLine label={t("invDetail.due")} value={formatDate(invoice.due_date)} />
        </div>
      </div>

      {/* items */}
      <div className="grid grid-cols-[1fr_60px_90px_90px] gap-3 pb-2 border-b border-[var(--ink)] text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">
        <span>{t("common.description")}</span>
        <span className="text-right">{t("common.qty")}</span>
        <span className="text-right">{t("common.rate")}</span>
        <span className="text-right">{t("common.amount")}</span>
      </div>
      {(invoice.items || []).map((it, i) => (
        <div key={i} className="grid grid-cols-[1fr_60px_90px_90px] gap-3 py-2.5 border-b border-[var(--border)] text-sm">
          <span className="text-[var(--ink)]">{it.description || "—"}</span>
          <span className="text-right tabular text-[var(--ink-muted)]">{Number(it.quantity)}</span>
          <span className="text-right tabular text-[var(--ink-muted)]">{formatMoney(it.rate, currency)}</span>
          <span className="text-right tabular text-[var(--ink)] font-medium">{formatMoney(it.amount, currency)}</span>
        </div>
      ))}

      {/* totals */}
      <div className="ml-auto w-full max-w-[260px] mt-5 space-y-2 text-sm">
        <TotalLine label={t("common.subtotal")} value={formatMoney(invoice.subtotal, currency)} />
        {Number(invoice.discount) > 0 && (
          <TotalLine label={t("common.discount")} value={`− ${formatMoney(invoice.discount, currency)}`} />
        )}
        <TotalLine label={t("invDetail.taxLine", { n: Number(invoice.tax_rate) })} value={formatMoney(invoice.tax_amount, currency)} />
        <div className="flex items-center justify-between pt-3 border-t border-[var(--ink)]">
          <span className="font-display font-semibold">{t("common.total")}</span>
          <span className="font-display text-xl font-semibold tabular text-[var(--accent-strong)]">
            {formatMoney(invoice.total, currency)}
          </span>
        </div>
      </div>

      {(invoice.notes || invoice.terms) && (
        <div className="mt-8 pt-5 border-t border-[var(--border)] space-y-3">
          {invoice.notes && (
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">{t("common.notes")}</div>
              <p className="text-sm text-[var(--ink)] whitespace-pre-line">{invoice.notes}</p>
            </div>
          )}
          {invoice.terms && (
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold mb-1">{t("common.terms")}</div>
              <p className="text-sm text-[var(--ink)] whitespace-pre-line">{invoice.terms}</p>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}

function MetaLine({ label, value }) {
  return (
    <div className="flex items-center justify-end gap-3">
      <span className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">{label}</span>
      <span className="tabular text-[var(--ink)] w-24 text-right">{value}</span>
    </div>
  );
}

function TotalLine({ label, value }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-[var(--ink-muted)]">{label}</span>
      <span className="tabular text-[var(--ink)]">{value}</span>
    </div>
  );
}

function PaymentCard({ invoice }) {
  const { t } = useLang();
  const { remove } = usePaymentMutations();
  const [modalOpen, setModalOpen] = useState(false);
  const currency = invoice.currency;
  const total = Number(invoice.total) || 0;
  const paid = Number(invoice.paid_amount) || 0;
  const balance = Number(invoice.balance) || 0;
  const pct = total > 0 ? Math.min(100, (paid / total) * 100) : 0;
  const payments = invoice.payments || [];

  // Shareable public link (POST /payments/online is get-or-create).
  const [shareLink, setShareLink] = useState(null);
  const [shareLoading, setShareLoading] = useState(false);
  const [shareErr, setShareErr] = useState("");
  const [linkCopied, setLinkCopied] = useState(false);
  const canShareOnline = balance > 0 && invoice.effective_status !== "draft" && currency === "IDR";
  const shareUrl = shareLink ? new URL(shareLink.url, window.location.origin).href : "";

  async function onDelete(p) {
    if (!window.confirm(t("payments.confirmDelete", { amount: formatMoney(p.amount, currency) }))) return;
    await remove.mutateAsync(p.id);
  }

  async function onShare() {
    if (shareLink || shareLoading) return;
    setShareLoading(true);
    setShareErr("");
    try {
      const res = await paymentsApi.createOnlineLink(invoice.id);
      setShareLink(res);
    } catch (e) {
      if (e.status !== 401) setShareErr(e.message || t("payments.saveFailed"));
    } finally {
      setShareLoading(false);
    }
  }

  async function copyShareLink() {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setLinkCopied(true);
      setTimeout(() => setLinkCopied(false), 1500);
    } catch {
      /* clipboard unavailable */
    }
  }

  return (
    <Card padding="lg">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="h-8 w-8 rounded-xl bg-[var(--accent-soft)] text-[var(--accent-strong)] flex items-center justify-center">
            <Wallet size={15} />
          </div>
          <CardTitle>{t("invDetail.payments")}</CardTitle>
        </div>
        {balance > 0 && (
          <Button variant="accent" size="sm" onClick={() => setModalOpen(true)}>
            <Plus size={13} /> {t("invDetail.recordPayment")}
          </Button>
        )}
      </div>

      <div className="space-y-1.5 text-sm mb-3">
        <TotalLine label={t("common.total")} value={formatMoney(total, currency)} />
        <div className="flex items-center justify-between">
          <span className="text-[var(--ink-muted)]">{t("invDetail.paidAmount")}</span>
          <span className="tabular font-semibold text-[var(--success)]">{formatMoney(paid, currency)}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-[var(--ink-muted)]">{t("invDetail.balanceDue")}</span>
          <span className={`tabular font-semibold ${balance > 0 ? "text-[var(--danger)]" : "text-[var(--ink)]"}`}>
            {formatMoney(balance, currency)}
          </span>
        </div>
      </div>

      <div className="h-1.5 w-full rounded-full bg-[var(--surface-2)] mb-4 overflow-hidden">
        <div className="h-full rounded-full bg-[var(--success)] transition-all" style={{ width: `${pct}%` }} />
      </div>

      {canShareOnline && !shareLink && (
        <Button variant="outline" size="sm" className="w-full mb-3" onClick={onShare} disabled={shareLoading}>
          {shareLoading ? <Loader2 size={13} className="animate-spin" /> : <Link2 size={13} />}
          {t("payments.shareLink")}
        </Button>
      )}
      {shareErr && !shareLink && <p className="text-xs text-[var(--danger)] mb-3">{shareErr}</p>}
      {shareLink && (
        <div className="flex items-center gap-2 rounded-2xl border border-[var(--border)] bg-[var(--surface-2)] px-3 py-2.5 mb-4">
          <span className="flex-1 min-w-0 text-xs text-[var(--ink)] truncate">{shareUrl}</span>
          <button type="button" onClick={copyShareLink} aria-label={t("payments.onlineCopy")}
            className="shrink-0 inline-flex items-center gap-1 text-[11px] font-semibold text-[var(--accent-strong)]">
            {linkCopied ? <Check size={12} /> : <Copy size={12} />}
            {linkCopied ? t("payments.onlineCopied") : t("payments.onlineCopy")}
          </button>
          <a href={shareUrl} target="_blank" rel="noreferrer" aria-label={t("payments.onlineOpen")}
            className="shrink-0 inline-flex items-center text-[var(--ink-muted)] hover:text-[var(--ink)]">
            <ExternalLink size={13} />
          </a>
        </div>
      )}

      {payments.length === 0 ? (
        <p className="text-xs text-[var(--ink-muted)]">{t("invDetail.noPayments")}</p>
      ) : (
        <ul className="divide-y divide-[var(--border)]">
          {payments.map((p) => (
            <li key={p.id} className="flex items-center gap-2 py-2.5 group">
              <div className="min-w-0 flex-1">
                <div className="text-xs font-semibold text-[var(--ink)]">{formatDate(p.paid_on)}</div>
                <div className="text-[11px] text-[var(--ink-muted)]">{p.method || "—"}{p.txn_id ? ` · ${p.txn_id}` : ""}</div>
              </div>
              <div className="text-sm font-semibold text-[var(--success)] tabular">{formatMoney(p.amount, currency)}</div>
              <button type="button"
                onClick={() => onDelete(p)}
                className="md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100 focus-visible:opacity-100 transition-opacity h-6 w-6 rounded-full flex items-center justify-center text-[var(--ink-muted)] hover:bg-[var(--surface-2)] hover:text-[var(--danger)]"
                aria-label={t("common.delete")}
              >
                <Trash2 size={12} />
              </button>
            </li>
          ))}
        </ul>
      )}

      <RecordPaymentModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        invoiceId={invoice.id}
        invoiceNumber={invoice.invoice_number}
        amount={balance}
        defaultEmail={invoice.client_email || ""}
      />
    </Card>
  );
}

function ClientCard({ invoice }) {
  const { t } = useLang();
  if (!invoice.client_id) return null;
  return (
    <Card padding="lg">
      <CardTitle className="mb-3">{t("common.client")}</CardTitle>
      <Link
        to={`/clients/${invoice.client_id}`}
        className="flex items-center gap-3 group"
      >
        <div className="h-10 w-10 rounded-full bg-[var(--accent-soft)] text-[var(--accent-strong)] flex items-center justify-center font-semibold">
          {invoice.client_name?.[0]?.toUpperCase() || "?"}
        </div>
        <div className="min-w-0">
          <div className="text-sm font-semibold text-[var(--ink)] group-hover:text-[var(--accent-strong)] truncate">
            {invoice.client_name}
          </div>
          <div className="text-xs text-[var(--ink-muted)] truncate">
            {invoice.client_email || invoice.client_company || t("invDetail.viewProfile")}
          </div>
        </div>
      </Link>
    </Card>
  );
}

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

function PaymentReminderCard({ invoiceId }) {
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

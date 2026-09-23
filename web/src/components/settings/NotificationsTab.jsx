import { useState } from "react";
import { BellRing, Check, Copy, KeyRound, Loader2, Plus, RefreshCw, RotateCw, Trash2, Webhook } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { QueryError } from "@/components/ui/QueryError";
import { Input } from "@/components/ui/Input";
import { useLang } from "@/context/LangContext";
import { useToast } from "@/context/UIContext";
import {
  useCreateNotificationEndpoint,
  useDeleteNotificationEndpoint,
  useNotificationDeliveries,
  useNotificationEndpoints,
  useRetryNotificationDelivery,
  useRotateNotificationSecret,
  useTestNotificationEndpoint,
  useUpdateNotificationEndpoint,
} from "@/hooks/useNotifications";

const EVENT_KEYS = ["invoice.created", "invoice.status_updated", "payment.created"];

function SecretNotice({ endpoint, onClose }) {
  const { t } = useLang();
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard?.writeText(endpoint.secret);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }
  return (
    <Card className="mb-5 border-[var(--accent)]/30 bg-[var(--accent-soft)]/30">
      <div className="flex items-start gap-3">
        <KeyRound size={18} className="mt-0.5 text-[var(--accent-strong)]" />
        <div className="min-w-0 flex-1">
          <div className="font-semibold text-[var(--ink)]">{t("notif.secretTitle")}</div>
          <p className="mt-1 text-sm text-[var(--ink-muted)]">{t("notif.secretDesc")}</p>
          <div className="mt-3 flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate rounded-xl bg-[var(--surface)] px-3 py-2 text-xs text-[var(--ink)]">{endpoint.secret}</code>
            <Button type="button" size="iconSm" variant="outline" onClick={copy} aria-label={t("notif.copy")}>
              {copied ? <Check size={13} /> : <Copy size={13} />}
            </Button>
          </div>
          <Button type="button" size="sm" variant="soft" className="mt-4" onClick={onClose}>{t("common.done")}</Button>
        </div>
      </div>
    </Card>
  );
}

function CreateEndpointForm({ onCreated }) {
  const { t } = useLang();
  const toast = useToast();
  const create = useCreateNotificationEndpoint();
  const [url, setUrl] = useState("");
  const [error, setError] = useState("");

  async function submit(event) {
    event.preventDefault();
    setError("");
    try {
      const endpoint = await create.mutateAsync({ target_url: url });
      setUrl("");
      onCreated(endpoint);
      toast.success(t("notif.created"));
    } catch (err) {
      if (err.status !== 401) setError(err.message || t("notif.saveFailed"));
    }
  }

  return (
    <Card className="mb-6">
      <div className="mb-4 flex items-center gap-2">
        <Plus size={17} className="text-[var(--accent-strong)]" />
        <h2 className="font-display text-lg font-semibold">{t("notif.addTitle")}</h2>
      </div>
      <form onSubmit={submit} className="flex flex-col gap-3 sm:flex-row">
        <Input value={url} onChange={(e) => setUrl(e.target.value)} placeholder={t("notif.urlPlaceholder")} type="url" required className="flex-1" />
        <Button type="submit" variant="accent" disabled={create.isPending}>
          {create.isPending ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
          {t("notif.add")}
        </Button>
      </form>
      {error && <p className="mt-2 text-sm text-[var(--danger)]">{error}</p>}
      <p className="mt-3 text-xs text-[var(--ink-muted)]">{t("notif.subscribeHint")}</p>
    </Card>
  );
}

function EndpointRow({ endpoint, onSecret }) {
  const { t } = useLang();
  const toast = useToast();
  const update = useUpdateNotificationEndpoint();
  const rotate = useRotateNotificationSecret();
  const remove = useDeleteNotificationEndpoint();
  const test = useTestNotificationEndpoint();
  const activeEvents = new Set((endpoint.events || "").split(",").map((e) => e.trim()).filter(Boolean));

  async function toggleEvent(eventType) {
    const next = new Set(activeEvents);
    if (activeEvents.size === 0) EVENT_KEYS.forEach((e) => next.add(e));
    if (next.has(eventType)) next.delete(eventType);
    else next.add(eventType);
    const payload = { events: next.size >= EVENT_KEYS.length ? "" : [...next].join(",") };
    try {
      await update.mutateAsync({ id: endpoint.id, payload });
    } catch (error) {
      if (error.status !== 401) toast.error(error.message || t("notif.saveFailed"));
    }
  }

  async function toggleActive() {
    try {
      await update.mutateAsync({ id: endpoint.id, payload: { is_active: !endpoint.is_active } });
    } catch (error) {
      if (error.status !== 401) toast.error(error.message || t("notif.saveFailed"));
    }
  }

  return (
    <Card className="mb-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <code className="min-w-0 flex-1 truncate text-xs text-[var(--ink)]" title={endpoint.target_url}>{endpoint.target_url}</code>
        <span className={`inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold ${endpoint.is_active ? "bg-[var(--success)]/12 text-[var(--success)]" : "bg-[var(--surface-2)] text-[var(--ink-muted)]"}`}>
          {endpoint.is_active ? t("notif.active") : t("notif.inactive")}
        </span>
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        {EVENT_KEYS.map((eventType) => {
          const on = activeEvents.size === 0 || activeEvents.has(eventType);
          return (
            <button
              key={eventType}
              type="button"
              onClick={() => toggleEvent(eventType)}
              className={`rounded-full px-3 py-1 text-[11px] font-semibold transition-colors ${on ? "bg-[var(--accent-soft)] text-[var(--accent-strong)]" : "bg-[var(--surface-2)] text-[var(--ink-muted)]"}`}
            >
              {eventType}
            </button>
          );
        })}
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button size="sm" variant="ghost" onClick={toggleActive} disabled={update.isPending}>{endpoint.is_active ? t("notif.disable") : t("notif.enable")}</Button>
        <Button size="sm" variant="outline" onClick={() => test.mutate(endpoint.id)} disabled={test.isPending}><BellRing size={13} />{t("notif.test")}</Button>
        <Button size="sm" variant="outline" onClick={async () => { try { onSecret(await rotate.mutateAsync(endpoint.id)); } catch (e) { if (e.status !== 401) toast.error(e.message || t("notif.saveFailed")); } }} disabled={rotate.isPending}><RotateCw size={13} />{t("notif.rotate")}</Button>
        <Button size="sm" variant="ghost" onClick={() => remove.mutate(endpoint.id)} disabled={remove.isPending} aria-label={t("common.delete")}><Trash2 size={13} /></Button>
      </div>
    </Card>
  );
}

function DeliveriesCard() {
  const { t } = useLang();
  const { data: deliveries = [], error } = useNotificationDeliveries();
  const retry = useRetryNotificationDelivery();
  const toast = useToast();
  if (error?.status !== 401 && error) return <QueryError error={error} />;
  return (
    <Card padding="none" className="overflow-hidden">
      <div className="flex items-center gap-2 border-b border-[var(--border)] p-4">
        <Webhook size={16} className="text-[var(--accent-strong)]" />
        <h2 className="font-display font-semibold">{t("notif.deliveries")}</h2>
      </div>
      <div className="max-h-[360px] overflow-auto">
        {deliveries.length ? deliveries.map((item) => (
          <div key={item.id} className="border-b border-[var(--border)] px-4 py-3 last:border-0">
            <div className="flex items-center justify-between gap-3">
              <code className="truncate text-xs text-[var(--ink)]">{item.event_type}</code>
              <span className="text-xs font-semibold text-[var(--accent-strong)]">{item.status}</span>
            </div>
            <div className="mt-1 flex items-center justify-between gap-2 text-xs text-[var(--ink-muted)]">
              <span className="truncate">{item.target_url} · {t("notif.attempt")} {item.attempt} · HTTP {item.resp_code || "-"}</span>
              {item.status !== "delivered" && (
                <Button size="iconSm" variant="ghost" onClick={async () => { try { await retry.mutateAsync(item.id); } catch (e) { if (e.status !== 401) toast.error(e.message || t("notif.saveFailed")); } }} disabled={retry.isPending} aria-label={t("notif.retry")}>
                  <RefreshCw size={13} />
                </Button>
              )}
            </div>
          </div>
        )) : <p className="p-6 text-sm text-[var(--ink-muted)]">{t("notif.noDeliveries")}</p>}
      </div>
    </Card>
  );
}

export default function NotificationsTab() {
  const { t } = useLang();
  const { data: endpoints = [], isLoading, error } = useNotificationEndpoints();
  const [secret, setSecret] = useState(null);
  if (isLoading) return <div className="flex items-center justify-center py-16 text-[var(--ink-muted)]"><Loader2 size={20} className="animate-spin" /></div>;
  if (error) return <QueryError error={error} />;
  return (
    <div className="max-w-2xl">
      <Card padding="lg" className="mb-6">
        <p className="text-sm text-[var(--ink-muted)]">{t("notif.desc")}</p>
      </Card>
      {secret?.secret && <SecretNotice endpoint={secret} onClose={() => setSecret(null)} />}
      <CreateEndpointForm onCreated={setSecret} />
      {endpoints.length
        ? endpoints.map((endpoint) => <EndpointRow key={endpoint.id} endpoint={endpoint} onSecret={setSecret} />)
        : <EmptyState icon={Webhook} title={t("notif.empty")} description={t("notif.emptyDesc")} />}
      <div className="mt-6"><DeliveriesCard /></div>
    </div>
  );
}

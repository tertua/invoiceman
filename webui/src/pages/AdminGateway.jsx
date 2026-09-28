import { useState } from "react";
import {
  Check,
  Copy,
  KeyRound,
  Loader2,
  Network,
  Plus,
  RotateCw,
  ShieldCheck,
} from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { QueryError } from "@/components/ui/QueryError";
import { GatewayDeliveries } from "@/components/gateway/GatewayDeliveries";
import { Input } from "@/components/ui/Input";
import { useLang } from "@/context/LangContext";
import { toast } from "sonner";
import {
  useCreateGatewayProject,
  useGatewayProjects,
  useGatewayStatus,
  useRotateGatewayKey,
  useRotateGatewaySecret,
  useUpdateGatewayProject,
} from "@/hooks/useGatewayAdmin";
import { cn } from "@/lib/utils";
import { StatusPill } from "@/components/gateway/StatusPill";
import { AdminTableSkeleton } from "@/components/admin/AdminTableSkeleton";

function SecretNotice({ project, onClose }) {
  const { t } = useLang();
  const [copied, setCopied] = useState("");
  async function copy(value, name) {
    await navigator.clipboard?.writeText(value);
    setCopied(name);
    window.setTimeout(() => setCopied(""), 1400);
  }
  return (
    <Card className="mb-5 border-[var(--accent)]/30 bg-[var(--accent-soft)]/30">
      <div className="flex items-start gap-3">
        <KeyRound size={18} className="mt-0.5 text-[var(--accent-strong)]" />
        <div className="min-w-0 flex-1">
          <div className="font-semibold text-[var(--ink)]">{t("gateway.credentialsTitle")}</div>
          <p className="mt-1 text-sm text-[var(--ink-muted)]">{t("gateway.credentialsDesc")}</p>
          {[
            ["apiKey", project.api_key],
            ["webhookSecret", project.webhook_secret],
          ].map(([name, value]) => (
            <div key={name} className="mt-3 flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-xl bg-[var(--surface)] px-3 py-2 text-xs text-[var(--ink)]">{value}</code>
              <Button type="button" size="iconSm" variant="outline" onClick={() => copy(value, name)} aria-label={t("gateway.copy")}>
                {copied === name ? <Check size={13} /> : <Copy size={13} />}
              </Button>
            </div>
          ))}
          <Button type="button" size="sm" variant="soft" className="mt-4" onClick={onClose}>{t("common.done")}</Button>
        </div>
      </div>
    </Card>
  );
}

function GatewayStatusBanner() {
  const { t } = useLang();
  const { data: gateways = [], isLoading } = useGatewayStatus();
  if (isLoading || !gateways.length) return null;
  return (
    <Card className="mb-6">
      <div className="mb-3 flex items-center gap-2">
        <Network size={16} className="text-[var(--accent-strong)]" />
        <h2 className="font-display text-base font-semibold">{t("gateway.availability")}</h2>
      </div>
      <div className="flex flex-wrap gap-2">
        {gateways.map((gw) => (
          <span
            key={gw.name}
            className={cn(
              "inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-xs font-semibold",
              gw.configured ? "bg-[var(--success)]/12 text-[var(--success)]" : "bg-[var(--surface-2)] text-[var(--ink-muted)]",
            )}
            title={gw.configured ? t("gateway.configured") : t("gateway.notConfigured")}
          >
            <code>{gw.name}</code>
            <span>·</span>
            <span>{gw.configured ? t("gateway.configured") : t("gateway.notConfigured")}</span>
            {gw.configured && <span>·</span>}
            {gw.configured && <span>{gw.sandbox ? t("gateway.sandbox") : t("gateway.live")}</span>}
          </span>
        ))}
      </div>
    </Card>
  );
}

function CreateProjectForm({ onCreated }) {
  const { t } = useLang();
  const create = useCreateGatewayProject();
  const { data: gateways = [] } = useGatewayStatus();
  const options = gateways.length ? gateways : [{ name: "midtrans", configured: true }, { name: "nowpayments", configured: false }];
  const [form, setForm] = useState({ slug: "", name: "", webhook_url: "", default_gateway: "midtrans" });
  const [error, setError] = useState("");
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }));

  async function submit(event) {
    event.preventDefault();
    setError("");
    try {
      const project = await create.mutateAsync(form);
      setForm({ slug: "", name: "", webhook_url: "", default_gateway: "midtrans" });
      onCreated(project);
      toast.success(t("gateway.created"));
    } catch (err) {
      if (err.status !== 401) setError(err.message || t("gateway.saveFailed"));
    }
  }

  return (
    <Card className="mb-6">
      <div className="mb-4 flex items-center gap-2">
        <Plus size={17} className="text-[var(--accent-strong)]" />
        <h2 className="font-display text-lg font-semibold">{t("gateway.addTitle")}</h2>
      </div>
      <form onSubmit={submit} className="grid gap-3 md:grid-cols-2">
        <Input value={form.slug} onChange={set("slug")} placeholder={t("gateway.slugPlaceholder")} required />
        <Input value={form.name} onChange={set("name")} placeholder={t("gateway.namePlaceholder")} required />
        <Input value={form.webhook_url} onChange={set("webhook_url")} placeholder={t("gateway.webhookPlaceholder")} type="url" required />
        <select
          value={form.default_gateway}
          onChange={set("default_gateway")}
          className="h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none transition-colors focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15"
          aria-label={t("gateway.gateway")}
        >
          {options.map((gw) => (
            <option key={gw.name} value={gw.name}>
              {gw.name}{gw.configured === false ? ` (${t("gateway.notConfigured")})` : ""}
            </option>
          ))}
        </select>
        {error && <p className="text-sm text-[var(--danger)] md:col-span-2">{error}</p>}
        <div className="md:col-span-2">
          <Button type="submit" variant="accent" disabled={create.isPending}>
            {create.isPending ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
            {t("gateway.addProject")}
          </Button>
        </div>
      </form>
    </Card>
  );
}

function ProjectsTable({ projects, onCredentials }) {
  const { t } = useLang();
  const rotateKey = useRotateGatewayKey();
  const rotateSecret = useRotateGatewaySecret();
  const update = useUpdateGatewayProject();
  async function rotate(mutation, slug) {
    try {
      onCredentials(await mutation.mutateAsync(slug));
    } catch (error) {
      if (error.status !== 401) toast.error(error.message || t("gateway.saveFailed"));
    }
  }
  async function toggle(project) {
    try {
      await update.mutateAsync({ slug: project.slug, payload: { is_active: !project.is_active } });
    } catch (error) {
      if (error.status !== 401) toast.error(error.message || t("gateway.saveFailed"));
    }
  }
  return (
    <Card padding="none" className="overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[900px] text-left">
          <thead className="bg-[var(--surface-2)] text-xs font-semibold uppercase tracking-wide text-[var(--ink-muted)]">
            <tr>
              <th className="px-4 py-3">{t("gateway.project")}</th>
              <th className="px-4 py-3">{t("gateway.gateway")}</th>
              <th className="px-4 py-3">{t("gateway.webhook")}</th>
              <th className="px-4 py-3">{t("gateway.status")}</th>
              <th className="px-4 py-3 text-right">{t("gateway.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {projects.map((project) => (
              <tr key={project.slug} className="border-t border-[var(--border)] align-top">
                <td className="px-4 py-4"><div className="font-medium text-[var(--ink)]">{project.name}</div><div className="mt-1 text-xs text-[var(--ink-muted)]">{project.slug}</div></td>
                <td className="px-4 py-4"><code className="rounded bg-[var(--surface-2)] px-2 py-1 text-xs">{project.default_gateway || "midtrans"}</code></td>
                <td className="max-w-[260px] truncate px-4 py-4 text-xs text-[var(--ink-muted)]" title={project.webhook_url}>{project.webhook_url}</td>
                <td className="px-4 py-4"><StatusPill active={project.is_active} t={t} /></td>
                <td className="px-4 py-4 text-right">
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button size="sm" variant="ghost" onClick={() => toggle(project)} disabled={update.isPending}>{project.is_active ? t("gateway.disable") : t("gateway.enable")}</Button>
                    <Button size="sm" variant="outline" onClick={() => rotate(rotateKey, project.slug)} disabled={rotateKey.isPending}><KeyRound size={13} />{t("gateway.rotateKey")}</Button>
                    <Button size="sm" variant="outline" onClick={() => rotate(rotateSecret, project.slug)} disabled={rotateSecret.isPending}><RotateCw size={13} />{t("gateway.rotateSecret")}</Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

export default function AdminGateway() {
  const { t } = useLang();
  const { data: projects = [], isLoading, error } = useGatewayProjects();
  const [credentials, setCredentials] = useState(null);
  if (isLoading) return <AdminTableSkeleton />;
  if (error) return <QueryError error={error} />;
  return (
    <div>
      <PageHeader title={t("gateway.title")} description={t("gateway.desc")} />
      {credentials && <SecretNotice project={credentials} onClose={() => setCredentials(null)} />}
      <GatewayStatusBanner />
      <CreateProjectForm onCreated={setCredentials} />
      {projects.length ? <ProjectsTable projects={projects} onCredentials={setCredentials} /> : <EmptyState icon={ShieldCheck} title={t("gateway.empty")} description={t("gateway.emptyDesc")} />}
      <GatewayDeliveries />
    </div>
  );
}

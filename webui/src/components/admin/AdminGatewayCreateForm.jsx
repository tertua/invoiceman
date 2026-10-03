import { useMemo, useState } from "react";
import { Loader2, Plus } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { GatewaySelect } from "@/components/gateway/GatewaySelect";
import { Input } from "@/components/ui/Input";
import { useLang } from "@/context/LangContext";
import { toast } from "sonner";
import { useCreateGatewayProject, useGatewayStatus } from "@/hooks/useGatewayAdmin";

export function AdminGatewayCreateForm({ onCreated }) {
  const { t } = useLang();
  const create = useCreateGatewayProject();
  const { data: gateways = [] } = useGatewayStatus();
  const options = useMemo(() => {
    const configured = gateways.filter((gw) => gw.configured !== false);
    return (configured.length ? configured : gateways).map((gw) => gw.name);
  }, [gateways]);
  const fallback = options[0] || "";
  const [form, setForm] = useState({ slug: "", name: "", webhook_url: "", default_gateway: "" });
  const [error, setError] = useState("");
  const chosen = options.includes(form.default_gateway) ? form.default_gateway : fallback;
  const setDefaultGateway = (event) => setForm((current) => ({ ...current, default_gateway: event.target.value }));
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }));

  async function submit(event) {
    event.preventDefault();
    setError("");
    try {
      const project = await create.mutateAsync({ ...form, default_gateway: chosen });
      setForm({ slug: "", name: "", webhook_url: "", default_gateway: "" });
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
        <GatewaySelect value={chosen} options={options} onChange={setDefaultGateway} lang={t} />
        {error && <p className="text-sm text-[var(--danger)] md:col-span-2">{error}</p>}
        <div className="md:col-span-2">
          <Button type="submit" variant="accent" disabled={create.isPending || !chosen}>
            {create.isPending ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
            {t("gateway.addProject")}
          </Button>
        </div>
      </form>
    </Card>
  );
}

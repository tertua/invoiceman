import { useMutation, useQueryClient } from "@tanstack/react-query";
import { KeyRound, RotateCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { adminGatewayApi } from "@/api/adminGateway";
import { StatusPill } from "@/components/gateway/StatusPill";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useLang } from "@/context/LangContext";
import { gatewayProjectsKey } from "@/hooks/useGatewayAdmin";
import {
  useRotateGatewayKey,
  useRotateGatewaySecret,
  useUpdateGatewayProject,
} from "@/hooks/useGatewayAdmin";

export function ProjectsTable({ projects, onCredentials }) {
  const { t } = useLang();
  const queryClient = useQueryClient();
  const rotateKey = useRotateGatewayKey();
  const rotateSecret = useRotateGatewaySecret();
  const update = useUpdateGatewayProject();
  const remove = useMutation({
    mutationFn: (slug) => adminGatewayApi.deleteGatewayProject(slug),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: gatewayProjectsKey }),
  });
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
  async function destroy(project) {
    if (!window.confirm(t("gateway.confirmDelete", { name: project.name }))) return;
    try {
      await remove.mutateAsync(project.slug);
      toast.success(t("gateway.deleted"));
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
                    <Button size="sm" variant="ghost" className="text-[var(--danger)] hover:bg-[var(--danger)]/10" onClick={() => destroy(project)} disabled={remove.isPending}><Trash2 size={13} />{t("common.delete")}</Button>
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

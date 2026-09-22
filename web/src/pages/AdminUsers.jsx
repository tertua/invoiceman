import { useState } from "react";
import { Loader2, Users } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { QueryError } from "@/components/ui/QueryError";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";
import { useToast } from "@/context/UIContext";
import { useAdminUsers, useUpdateUserRole } from "@/hooks/useAdminUsers";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";

const ROLES = ["user", "moderator"];

function userStatus(status, t) {
  return status === 1 ? t("admin.statusActive") : t("admin.statusBlocked");
}

function RoleBadge({ role }) {
  const { t } = useLang();
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-semibold capitalize",
        role === "admin"
          ? "bg-[var(--accent-soft)] text-[var(--accent-strong)]"
          : "bg-[var(--surface-2)] text-[var(--ink-muted)]",
      )}
    >
      {t(`admin.role.${role}`)}
    </span>
  );
}

function UserRow({ account, currentUserId }) {
  const { t } = useLang();
  const toast = useToast();
  const updateRole = useUpdateUserRole();
  const [role, setRole] = useState(account.role);
  const isCurrentUser = account.id === currentUserId;
  const isAdmin = account.role === "admin";
  const canEdit = !isCurrentUser && !isAdmin;

  async function onRoleChange(event) {
    const nextRole = event.target.value;
    setRole(nextRole);
    try {
      await updateRole.mutateAsync({ id: account.id, role: nextRole });
      toast.success(t("admin.roleUpdated"));
    } catch (error) {
      setRole(account.role);
      if (error?.status !== 401) toast.error(t("admin.roleUpdateFailed"), error?.message);
    }
  }

  return (
    <tr className="border-t border-[var(--border)]">
      <td className="px-4 py-4">
        <div className="font-medium text-[var(--ink)]">{account.name}</div>
        <div className="mt-0.5 text-xs text-[var(--ink-muted)]">{account.email}</div>
      </td>
      <td className="px-4 py-4"><RoleBadge role={account.role} /></td>
      <td className="px-4 py-4 text-sm text-[var(--ink-muted)]">{userStatus(account.status, t)}</td>
      <td className="px-4 py-4 text-sm text-[var(--ink-muted)] tabular">{formatDate(account.created_at)}</td>
      <td className="px-4 py-4 text-right">
        {canEdit ? (
          <select
            aria-label={t("admin.changeRoleFor", { name: account.name })}
            value={role}
            onChange={onRoleChange}
            disabled={updateRole.isPending}
            className="h-9 rounded-full border border-[var(--border)] bg-[var(--surface)] px-3 text-xs font-medium text-[var(--ink)] outline-none focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15 disabled:opacity-60"
          >
            {ROLES.map((value) => <option key={value} value={value}>{t(`admin.role.${value}`)}</option>)}
          </select>
        ) : (
          <span className="text-xs text-[var(--ink-muted)]">
            {isCurrentUser ? t("admin.you") : t("admin.protected")}
          </span>
        )}
      </td>
    </tr>
  );
}

export default function AdminUsers() {
  const { user } = useAuth();
  const { t } = useLang();
  const { data: users, isLoading, error } = useAdminUsers();

  if (isLoading) {
    return <div className="flex items-center justify-center py-24 text-[var(--ink-muted)]"><Loader2 size={20} className="animate-spin" /></div>;
  }

  if (error) {
    return <QueryError error={error} />;
  }

  return (
    <div>
      <PageHeader title={t("admin.title")} description={t("admin.desc")} />
      {!users?.length ? (
        <EmptyState icon={Users} title={t("admin.empty")} description={t("admin.emptyDesc")} />
      ) : (
        <Card padding="none" className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[700px] text-left">
              <thead className="bg-[var(--surface-2)] text-xs font-semibold uppercase tracking-wide text-[var(--ink-muted)]">
                <tr>
                  <th className="px-4 py-3">{t("admin.user")}</th>
                  <th className="px-4 py-3">{t("admin.role")}</th>
                  <th className="px-4 py-3">{t("admin.status")}</th>
                  <th className="px-4 py-3">{t("admin.joined")}</th>
                  <th className="px-4 py-3 text-right">{t("admin.action")}</th>
                </tr>
              </thead>
              <tbody>
                {users.map((account) => (
                  <UserRow key={account.id} account={account} currentUserId={user?.id} />
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  );
}

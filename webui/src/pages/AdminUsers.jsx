import { Users } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";import { QueryError } from "@/components/ui/QueryError";
import { AdminTableSkeleton } from "@/components/admin/AdminTableSkeleton";
import { useLang } from "@/context/LangContext";
import { useAdminUsers } from "@/hooks/useAdminUsers";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";

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

function UserRow({ account }) {
  const { t } = useLang();
  return (
    <tr className="border-t border-[var(--border)]">
      <td className="px-4 py-4">
        <div className="font-medium text-[var(--ink)]">{account.name}</div>
        <div className="mt-0.5 text-xs text-[var(--ink-muted)]">{account.email}</div>
      </td>
      <td className="px-4 py-4"><RoleBadge role={account.role} /></td>
      <td className="px-4 py-4 text-sm text-[var(--ink-muted)]">{userStatus(account.status, t)}</td>
      <td className="px-4 py-4 text-sm text-[var(--ink-muted)] tabular">{formatDate(account.created_at)}</td>
    </tr>
  );
}

export default function AdminUsers() {
  const { t } = useLang();
  const { data: users, isLoading, error } = useAdminUsers();

  if (isLoading) {
    return <AdminTableSkeleton rows={4} />;
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
                </tr>
              </thead>
              <tbody>
                {users.map((account) => (
                  <UserRow key={account.id} account={account} />
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  );
}

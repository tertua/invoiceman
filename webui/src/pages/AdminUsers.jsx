import { ShieldCheck, Users, UserX } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { AdminTableSkeleton } from "@/components/admin/AdminTableSkeleton";
import { AdminUsersTable } from "@/components/admin/AdminUsersTable";
import { PageStat, PageStatStrip } from "@/components/admin/PageStatStrip";
import { EmptyState } from "@/components/ui/EmptyState";
import { QueryError } from "@/components/ui/QueryError";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";
import { useAdminUsers } from "@/hooks/useAdminUsers";

export default function AdminUsers() {
  const { user } = useAuth();
  const { t } = useLang();
  const { data: users, isLoading, error } = useAdminUsers();

  if (isLoading) {
    return <AdminTableSkeleton rows={4} columns={5} minWidthClassName="min-w-[700px]" />;
  }

  if (error) {
    return <QueryError error={error} />;
  }

  const admins = users?.filter((account) => account.role === "admin").length || 0;
  const blocked = users?.filter((account) => account.status === 0).length || 0;

  return (
    <div>
      <PageHeader title={t("admin.title")} description={t("admin.desc")} />
      {!users?.length ? (
        <EmptyState icon={Users} title={t("admin.empty")} description={t("admin.emptyDesc")} />
      ) : (
        <>
          <PageStatStrip>
            <PageStat label={t("admin.statUsers")} value={users.length} icon={Users} tone="accent" />
            <PageStat label={t("admin.statAdmins")} value={admins} icon={ShieldCheck} tone="success" />
            <PageStat label={t("admin.statBlocked")} value={blocked} icon={UserX} tone={blocked ? "danger" : "neutral"} />
          </PageStatStrip>
          <AdminUsersTable users={users} currentUserId={user?.id} />
        </>
      )}
    </div>
  );
}

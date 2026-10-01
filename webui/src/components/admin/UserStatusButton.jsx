import { toast } from "sonner";
import { useLang } from "@/context/LangContext";
import { useUpdateUserStatus } from "@/hooks/useAdminUserStatus";
import { cn } from "@/lib/utils";

// Block/unblock toggle for one admin user row. Hidden for the current admin
// (self-change is refused server-side too).
export function UserStatusButton({ account }) {
  const { t } = useLang();
  const updateStatus = useUpdateUserStatus();
  const isBlocked = account.status === 0;

  async function handleToggle() {
    const next = isBlocked ? 1 : 0;
    try {
      await updateStatus.mutateAsync({ id: account.id, status: next });
      toast.success(next === 0 ? t("admin.userBlocked") : t("admin.userUnblocked"));
    } catch (error) {
      if (error?.status !== 401) {
        toast.error(t("admin.statusUpdateFailed"), { description: error?.message });
      }
    }
  }

  return (
    <button
      type="button"
      onClick={handleToggle}
      disabled={updateStatus.isPending}
      className={cn(
        "h-9 rounded-full border px-3 text-xs font-medium outline-none transition-colors focus:ring-2 focus:ring-[var(--accent)]/15 disabled:opacity-50 disabled:pointer-events-none",
        isBlocked
          ? "border-[var(--accent)]/30 text-[var(--accent-strong)] hover:bg-[var(--accent-soft)]"
          : "border-[var(--border)] text-[var(--ink-muted)] hover:bg-[var(--surface-2)]",
      )}
    >
      {isBlocked ? t("admin.unblock") : t("admin.block")}
    </button>
  );
}

import { AlertTriangle } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { EmptyState } from "@/components/ui/EmptyState";
import { Button } from "@/components/ui/Button";
import { useLang } from "@/context/LangContext";

// QueryError renders a failed-query state with a retry affordance.
// 401s return null: the global auth-expired broadcast already bounces the
// user to /login, so inline text would only flash before the redirect.
// `invalidate` optionally narrows refetch to specific query keys.
export function QueryError({ error, invalidate }) {
  const { t } = useLang();
  const queryClient = useQueryClient();
  if (!error || error.status === 401) return null;
  return (
    <EmptyState
      icon={AlertTriangle}
      title={t("common.loadFailed")}
      description={error.message}
      action={
        <Button
          variant="outline"
          onClick={() =>
            invalidate
              ? queryClient.invalidateQueries({ queryKey: invalidate })
              : queryClient.refetchQueries()
          }
        >
          {t("common.retry")}
        </Button>
      }
    />
  );
}

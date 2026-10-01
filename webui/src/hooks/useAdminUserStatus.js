import { useMutation, useQueryClient } from "@tanstack/react-query";
import { adminUsersApi } from "@/api/adminUsers";
import { adminUsersKey } from "./useAdminUsers";

// Blocks/unblocks an account and refreshes the users table on success.
export function useUpdateUserStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }) => adminUsersApi.updateUserStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: adminUsersKey }),
  });
}

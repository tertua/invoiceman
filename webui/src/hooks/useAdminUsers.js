import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/api/admin";

export const adminUsersKey = ["admin", "users"];

export function useAdminUsers() {
  return useQuery({ queryKey: adminUsersKey, queryFn: adminApi.listUsers });
}

export function useUpdateUserRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, role }) => adminApi.updateUserRole(id, role),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: adminUsersKey }),
  });
}

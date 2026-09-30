import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { orgsApi } from "@/api/orgs";

export const orgsKey = (...parts) => ["orgs", ...parts];

export function useOrgMe() {
  return useQuery({ queryKey: orgsKey("me"), queryFn: () => orgsApi.me() });
}

export function useOrgMembers() {
  return useQuery({ queryKey: orgsKey("members"), queryFn: () => orgsApi.members() });
}

export function useOrgInvites() {
  return useQuery({ queryKey: orgsKey("invites"), queryFn: () => orgsApi.invites() });
}

export function useActivateOrg() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id) => orgsApi.activate(id),
    onSuccess: () => qc.clear(), // active-org swap invalidates everything (AuthContext pattern)
  });
}

export function useAcceptInvite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (token) => orgsApi.accept({ token }),
    // A join swaps membership, role and data scope at once: clear everything (same as the activate swap) so stale per-org queries never leak.
    onSuccess: () => qc.clear(),
  });
}

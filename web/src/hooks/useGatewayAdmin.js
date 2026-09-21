import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/api/admin";

export const gatewayProjectsKey = ["admin", "gateway", "projects"];
export const gatewayTransactionsKey = ["admin", "gateway", "transactions"];
export const gatewayDeliveriesKey = ["admin", "gateway", "deliveries"];

export function useGatewayProjects() {
  return useQuery({ queryKey: gatewayProjectsKey, queryFn: adminApi.listGatewayProjects });
}

export function useGatewayTransactions() {
  return useQuery({ queryKey: gatewayTransactionsKey, queryFn: adminApi.listGatewayTransactions });
}

export function useGatewayDeliveries() {
  return useQuery({ queryKey: gatewayDeliveriesKey, queryFn: adminApi.listGatewayDeliveries });
}

function useGatewayMutation(mutationFn, keys) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => keys.forEach((key) => queryClient.invalidateQueries({ queryKey: key })),
  });
}

export function useCreateGatewayProject() {
  return useGatewayMutation(adminApi.createGatewayProject, [gatewayProjectsKey]);
}

export function useUpdateGatewayProject() {
  return useGatewayMutation(({ slug, payload }) => adminApi.updateGatewayProject(slug, payload), [gatewayProjectsKey]);
}

export function useRotateGatewayKey() {
  return useGatewayMutation(adminApi.rotateGatewayKey, [gatewayProjectsKey]);
}

export function useRotateGatewaySecret() {
  return useGatewayMutation(adminApi.rotateGatewaySecret, [gatewayProjectsKey]);
}

export function useRetryGatewayDelivery() {
  return useGatewayMutation(adminApi.retryGatewayDelivery, [gatewayDeliveriesKey]);
}

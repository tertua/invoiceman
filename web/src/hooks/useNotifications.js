import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "@/api/notifications";

export const notificationEndpointsKey = ["settings", "notifications", "endpoints"];
export const notificationDeliveriesKey = ["settings", "notifications", "deliveries"];

export function useNotificationEndpoints() {
  return useQuery({ queryKey: notificationEndpointsKey, queryFn: notificationsApi.listEndpoints });
}

export function useNotificationDeliveries() {
  return useQuery({ queryKey: notificationDeliveriesKey, queryFn: () => notificationsApi.listDeliveries() });
}

function useNotificationMutation(mutationFn, keys) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => keys.forEach((key) => queryClient.invalidateQueries({ queryKey: key })),
  });
}

export function useCreateNotificationEndpoint() {
  return useNotificationMutation(notificationsApi.createEndpoint, [notificationEndpointsKey]);
}

export function useUpdateNotificationEndpoint() {
  return useNotificationMutation(
    ({ id, payload }) => notificationsApi.updateEndpoint(id, payload),
    [notificationEndpointsKey],
  );
}

export function useDeleteNotificationEndpoint() {
  return useNotificationMutation(notificationsApi.deleteEndpoint, [
    notificationEndpointsKey,
    notificationDeliveriesKey,
  ]);
}

export function useRotateNotificationSecret() {
  return useNotificationMutation(notificationsApi.rotateSecret, [notificationEndpointsKey]);
}

export function useTestNotificationEndpoint() {
  return useNotificationMutation(notificationsApi.testEndpoint, [notificationDeliveriesKey]);
}

export function useRetryNotificationDelivery() {
  return useNotificationMutation(notificationsApi.retryDelivery, [notificationDeliveriesKey]);
}

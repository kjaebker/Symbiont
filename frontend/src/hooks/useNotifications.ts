import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getNotificationTargets,
  upsertNotificationTarget,
  deleteNotificationTarget,
  testNotifications,
} from '@/api/client'
import type { NotificationTarget } from '@/api/types'

export function useNotificationTargets() {
  return useQuery({
    queryKey: ['notifications', 'targets'],
    queryFn: getNotificationTargets,
    staleTime: 30_000,
  })
}

export function useUpsertNotificationTarget() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (target: Omit<NotificationTarget, 'id'> & { id?: number }) =>
      upsertNotificationTarget(target),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications', 'targets'] })
    },
  })
}

export function useDeleteNotificationTarget() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteNotificationTarget(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications', 'targets'] })
    },
  })
}

export function useTestNotifications() {
  return useMutation({
    mutationFn: testNotifications,
  })
}

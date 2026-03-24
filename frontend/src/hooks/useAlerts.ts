import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getAlerts, createAlert, updateAlert, deleteAlert, getAlertEvents } from '@/api/client'
import type { AlertRule } from '@/api/types'

export function useAlertEvents(params?: { rule_id?: number; active_only?: boolean; limit?: number }) {
  return useQuery({
    queryKey: ['alerts', 'events', params],
    queryFn: () => getAlertEvents(params),
    staleTime: 10_000,
  })
}

export function useAlerts() {
  return useQuery({
    queryKey: ['alerts'],
    queryFn: getAlerts,
    staleTime: 10_000,
  })
}

export function useCreateAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (rule: Omit<AlertRule, 'id' | 'created_at'>) => createAlert(rule),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
  })
}

export function useUpdateAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, rule }: { id: number; rule: Partial<AlertRule> }) => updateAlert(id, rule),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
  })
}

export function useDeleteAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
  })
}

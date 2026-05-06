import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getDashboardLayout,
  replaceDashboardLayout,
  addDashboardItem,
  removeDashboardItem,
} from '@/api/client'
import { qk } from '@/api/queryKeys'
import type { DashboardItem } from '@/api/types'

export function useDashboardLayout() {
  return useQuery({
    queryKey: qk.dashboard.layout,
    queryFn: getDashboardLayout,
    staleTime: 10_000,
  })
}

export function useReplaceDashboardLayout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (items: Omit<DashboardItem, 'id' | 'sort_order'>[]) => replaceDashboardLayout(items),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.dashboard.layout })
    },
  })
}

export function useAddDashboardItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (item: Omit<DashboardItem, 'id' | 'sort_order'>) => addDashboardItem(item),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.dashboard.layout })
    },
  })
}

export function useRemoveDashboardItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => removeDashboardItem(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.dashboard.layout })
    },
  })
}

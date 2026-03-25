import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getDashboardLayout,
  replaceDashboardLayout,
  addDashboardItem,
  removeDashboardItem,
} from '@/api/client'
import type { DashboardItem } from '@/api/types'

export function useDashboardLayout() {
  return useQuery({
    queryKey: ['dashboardLayout'],
    queryFn: getDashboardLayout,
    staleTime: 10_000,
  })
}

export function useReplaceDashboardLayout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (items: Omit<DashboardItem, 'id' | 'sort_order'>[]) => replaceDashboardLayout(items),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboardLayout'] })
    },
  })
}

export function useAddDashboardItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (item: Omit<DashboardItem, 'id' | 'sort_order'>) => addDashboardItem(item),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboardLayout'] })
    },
  })
}

export function useRemoveDashboardItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => removeDashboardItem(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboardLayout'] })
    },
  })
}

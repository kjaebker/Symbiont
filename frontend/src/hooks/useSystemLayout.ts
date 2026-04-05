import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getSystemLayout, saveSystemLayout } from '@/api/client'
import type { SystemLayout } from '@/api/types'

export function useSystemLayout() {
  return useQuery({
    queryKey: ['systemLayout'],
    queryFn: getSystemLayout,
    staleTime: 30_000,
  })
}

export function useSaveSystemLayout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (layout: SystemLayout) => saveSystemLayout(layout),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['systemLayout'] })
    },
  })
}

import { useQuery } from '@tanstack/react-query'
import { getSystemStatus, getSystemLog } from '@/api/client'

export function useSystemStatus() {
  return useQuery({
    queryKey: ['system'],
    queryFn: getSystemStatus,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

export function useSystemLog(params?: { limit?: number; service?: string }) {
  return useQuery({
    queryKey: ['system', 'log', params],
    queryFn: () => getSystemLog(params),
    staleTime: 30_000,
    refetchInterval: false,
  })
}

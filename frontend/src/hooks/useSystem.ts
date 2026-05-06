import { useQuery } from '@tanstack/react-query'
import { getSystemStatus, getSystemLog } from '@/api/client'
import { qk } from '@/api/queryKeys'

export function useSystemStatus() {
  return useQuery({
    queryKey: qk.system.status,
    queryFn: getSystemStatus,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

export function useSystemLog(params?: { limit?: number; service?: string }) {
  return useQuery({
    queryKey: qk.system.log(params),
    queryFn: () => getSystemLog(params),
    staleTime: 30_000,
    refetchInterval: false,
  })
}

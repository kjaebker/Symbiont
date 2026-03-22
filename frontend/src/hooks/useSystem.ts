import { useQuery } from '@tanstack/react-query'
import { getSystemStatus } from '@/api/client'

export function useSystemStatus() {
  return useQuery({
    queryKey: ['system'],
    queryFn: getSystemStatus,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

import { useQuery } from '@tanstack/react-query'
import { listAuditEvents, getEventBusStats } from '@/api/client'
import type { AuditEventListParams } from '@/api/client'

export function useAuditEvents(params?: AuditEventListParams) {
  return useQuery({
    queryKey: ['audit-events', params],
    queryFn: () => listAuditEvents(params),
    staleTime: 15_000,
  })
}

export function useEventBusStats() {
  return useQuery({
    queryKey: ['event-bus-stats'],
    queryFn: getEventBusStats,
    refetchInterval: 10_000, // poll every 10s for live queue depths
    staleTime: 5_000,
  })
}

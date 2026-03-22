import { useQuery } from '@tanstack/react-query'
import { getProbes, getProbeHistory } from '@/api/client'

export function useProbes() {
  return useQuery({
    queryKey: ['probes'],
    queryFn: getProbes,
    staleTime: 10_000,
    refetchInterval: false,
  })
}

export function useProbeHistory(
  name: string | null,
  params?: { from?: string; to?: string; interval?: string },
) {
  return useQuery({
    queryKey: ['probeHistory', name, params],
    queryFn: () => getProbeHistory(name!, params),
    enabled: !!name,
    staleTime: 30_000,
  })
}

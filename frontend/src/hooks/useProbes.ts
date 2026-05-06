import { useQuery } from '@tanstack/react-query'
import { getProbes, getProbeHistory } from '@/api/client'
import { qk } from '@/api/queryKeys'

export function useProbes() {
  return useQuery({
    queryKey: qk.probes.all,
    queryFn: getProbes,
    staleTime: 10_000,
    refetchInterval: false,
    notifyOnChangeProps: ['data', 'error'],
  })
}

export function useProbeHistory(
  name: string | null,
  params?: { from?: string; to?: string; interval?: string },
) {
  return useQuery({
    queryKey: qk.probes.history(name ?? '', params),
    queryFn: () => getProbeHistory(name!, params),
    enabled: !!name,
    staleTime: 30_000,
  })
}

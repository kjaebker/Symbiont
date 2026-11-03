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

// Sparklines show the last 2h in 5m buckets, so refetching more often than
// once per bucket returns the same data. These are deliberately NOT refreshed
// by SSE (see useSSE) — that fired every 10s per sparkline.
const SPARKLINE_WINDOW_MS = 2 * 60 * 60 * 1000
const SPARKLINE_REFRESH_MS = 5 * 60_000

export function useProbeSparkline(name: string | null, enabled = true) {
  return useQuery({
    queryKey: qk.probes.sparkline(name),
    // Window start is computed per fetch so interval refetches slide forward.
    queryFn: () =>
      getProbeHistory(name!, {
        from: new Date(Date.now() - SPARKLINE_WINDOW_MS).toISOString(),
        interval: '5m',
      }),
    enabled: !!name && enabled,
    staleTime: SPARKLINE_REFRESH_MS,
    refetchInterval: SPARKLINE_REFRESH_MS,
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

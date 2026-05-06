import { apiFetch } from './_fetch'
import type { Probe, ProbeHistory, ProbeConfig } from './types'

export function getProbes() {
  return apiFetch<{ probes: Probe[]; polled_at: string }>('/api/probes')
}

export function getProbeHistory(
  name: string,
  params?: { from?: string; to?: string; interval?: string },
) {
  const search = new URLSearchParams()
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  if (params?.interval) search.set('interval', params.interval)
  const qs = search.toString()
  return apiFetch<ProbeHistory>(
    `/api/probes/${encodeURIComponent(name)}/history${qs ? `?${qs}` : ''}`,
  )
}

export function getProbeConfigs() {
  return apiFetch<{ configs: ProbeConfig[] }>('/api/config/probes')
}

export function updateProbeConfig(name: string, config: Partial<ProbeConfig>) {
  return apiFetch<ProbeConfig>(`/api/config/probes/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

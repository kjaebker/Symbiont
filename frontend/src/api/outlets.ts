import { apiFetch } from './_fetch'
import type { Outlet, OutletHistory, OutletConfig } from './types'

export type { Outlet }

export function getOutlets() {
  return apiFetch<{ outlets: Outlet[]; polled_at: string }>('/api/outlets')
}

export function setOutletState(id: string, state: 'ON' | 'OFF' | 'AUTO') {
  return apiFetch<{ outlet: Outlet }>(`/api/outlets/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ state }),
  })
}

export function getOutletHistory(
  id: string,
  params?: { from?: string; to?: string; interval?: string },
) {
  const search = new URLSearchParams()
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  if (params?.interval) search.set('interval', params.interval)
  const qs = search.toString()
  return apiFetch<OutletHistory>(
    `/api/outlets/${encodeURIComponent(id)}/history${qs ? `?${qs}` : ''}`,
  )
}

export function getOutletConfigs() {
  return apiFetch<{ configs: OutletConfig[] }>('/api/config/outlets')
}

export function updateOutletConfig(id: string, config: Partial<OutletConfig>) {
  return apiFetch<OutletConfig>(`/api/config/outlets/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

import { apiFetch } from './_fetch'
import type { SystemStatus, SystemLogLine } from './types'

export function getSystemStatus() {
  return apiFetch<SystemStatus>('/api/system')
}

export function getSystemLog(params?: { limit?: number; service?: string }) {
  const search = new URLSearchParams()
  if (params?.limit) search.set('limit', String(params.limit))
  if (params?.service) search.set('service', params.service)
  const qs = search.toString()
  return apiFetch<{ lines: SystemLogLine[] }>(`/api/system/log${qs ? `?${qs}` : ''}`)
}

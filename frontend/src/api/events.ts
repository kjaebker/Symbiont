import { apiFetch } from './_fetch'

export interface AuditEvent {
  id: number
  ts: string
  kind: string
  payload_json: string
  correlation_id?: string
}

export interface SubscriberStat {
  name: string
  queue_depth: number
  published: number
  dropped: number
  last_error?: string
}

export interface AuditEventListParams {
  kind?: string
  since?: string // RFC3339
  limit?: number
  correlation_id?: string
  initiated_by?: string
}

export function listAuditEvents(params?: AuditEventListParams) {
  const q = new URLSearchParams()
  if (params?.kind) q.set('kind', params.kind)
  if (params?.since) q.set('since', params.since)
  if (params?.limit) q.set('limit', String(params.limit))
  if (params?.correlation_id) q.set('correlation_id', params.correlation_id)
  if (params?.initiated_by) q.set('initiated_by', params.initiated_by)
  const qs = q.toString()
  return apiFetch<{ events: AuditEvent[] }>(`/api/events${qs ? `?${qs}` : ''}`)
}

export function getEventBusStats() {
  return apiFetch<{ subscribers: SubscriberStat[] }>('/api/events/stats')
}

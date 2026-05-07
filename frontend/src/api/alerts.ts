import { apiFetch } from './_fetch'
import type { AlertRule, AlertEvent } from './types'

export function getAlerts() {
  return apiFetch<{ rules: AlertRule[] }>('/api/alerts')
}

export function createAlert(rule: Omit<AlertRule, 'id' | 'created_at'>) {
  return apiFetch<AlertRule>('/api/alerts', {
    method: 'POST',
    body: JSON.stringify(rule),
  })
}

export function updateAlert(id: number, rule: Partial<AlertRule>) {
  return apiFetch<AlertRule>(`/api/alerts/${id}`, {
    method: 'PUT',
    body: JSON.stringify(rule),
  })
}

export function deleteAlert(id: number) {
  return apiFetch<void>(`/api/alerts/${id}`, { method: 'DELETE' })
}

export function getAlertEvents(params?: { rule_id?: number; active_only?: boolean; limit?: number }) {
  const search = new URLSearchParams()
  if (params?.rule_id) search.set('rule_id', String(params.rule_id))
  if (params?.active_only) search.set('active_only', '1')
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ events: AlertEvent[] }>(`/api/alerts/events${qs ? `?${qs}` : ''}`)
}

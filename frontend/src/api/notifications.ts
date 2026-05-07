import { apiFetch } from './_fetch'
import type { NotificationTarget, NotificationTestResult } from './types'

export function getNotificationTargets() {
  return apiFetch<{ targets: NotificationTarget[] }>('/api/notifications/targets')
}

export function upsertNotificationTarget(target: Omit<NotificationTarget, 'id'> & { id?: number }) {
  return apiFetch<NotificationTarget>('/api/notifications/targets', {
    method: 'POST',
    body: JSON.stringify(target),
  })
}

export function deleteNotificationTarget(id: number) {
  return apiFetch<void>(`/api/notifications/targets/${id}`, { method: 'DELETE' })
}

export function testNotifications() {
  return apiFetch<{ results: NotificationTestResult[] }>('/api/notifications/test', { method: 'POST' })
}

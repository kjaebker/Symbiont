import { apiFetch } from './_fetch'
import type {
  MaintenanceTask,
  MaintenanceFrequency,
  MaintenanceLog,
  DueItem,
} from './types'

export type {
  MaintenanceTask, MaintenanceFrequency, MaintenanceLog, DueItem,
}

export function getMaintenanceTasks() {
  return apiFetch<{ tasks: MaintenanceTask[] }>('/api/maintenance/tasks')
}

export function createMaintenanceTask(data: {
  name: string; description?: string; frequency: MaintenanceFrequency;
  interval_days?: number; day_of_week?: number; enabled?: boolean
}) {
  return apiFetch<{ task: MaintenanceTask }>('/api/maintenance/tasks', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateMaintenanceTask(id: number, data: {
  name: string; description?: string; frequency: MaintenanceFrequency;
  interval_days?: number; day_of_week?: number; enabled: boolean
}) {
  return apiFetch<{ task: MaintenanceTask }>(`/api/maintenance/tasks/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteMaintenanceTask(id: number) {
  return apiFetch<{ ok: boolean }>(`/api/maintenance/tasks/${id}`, { method: 'DELETE' })
}

export function completeMaintenanceTask(id: number, data?: { notes?: string; completed_at?: string }) {
  return apiFetch<{ log_id: number }>(`/api/maintenance/tasks/${id}/complete`, {
    method: 'POST',
    body: JSON.stringify(data ?? {}),
  })
}

export function getMaintenanceLogs(taskId: number, limit?: number) {
  const qs = limit ? `?limit=${limit}` : ''
  return apiFetch<{ logs: MaintenanceLog[] }>(`/api/maintenance/tasks/${taskId}/logs${qs}`)
}

export function getDueItems() {
  return apiFetch<{ items: DueItem[] }>('/api/tasks/due')
}

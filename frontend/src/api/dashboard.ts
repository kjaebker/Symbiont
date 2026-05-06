import { apiFetch } from './_fetch'
import type { DashboardItem } from './types'

export function getDashboardLayout() {
  return apiFetch<{ items: DashboardItem[] }>('/api/dashboard')
}

export function replaceDashboardLayout(items: Omit<DashboardItem, 'id' | 'sort_order'>[]) {
  return apiFetch<{ items: DashboardItem[] }>('/api/dashboard', {
    method: 'PUT',
    body: JSON.stringify({ items }),
  })
}

export function addDashboardItem(item: Omit<DashboardItem, 'id' | 'sort_order'>) {
  return apiFetch<DashboardItem>('/api/dashboard', {
    method: 'POST',
    body: JSON.stringify(item),
  })
}

export function removeDashboardItem(id: number) {
  return apiFetch<void>(`/api/dashboard/${id}`, { method: 'DELETE' })
}

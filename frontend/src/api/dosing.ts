import { apiFetch } from './_fetch'
import type {
  DosingProduct,
  DosingProductType,
  DosingSchedule,
  DosingFrequency,
  DosingLog,
} from './types'

export type {
  DosingProduct, DosingProductType, DosingSchedule, DosingFrequency, DosingLog,
}

export function getDosingProducts() {
  return apiFetch<{ products: DosingProduct[] }>('/api/dosing/products')
}

export function createDosingProduct(data: {
  brand: string; name: string; type: DosingProductType; unit: string; notes?: string
}) {
  return apiFetch<{ product: DosingProduct }>('/api/dosing/products', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateDosingProduct(id: number, data: {
  brand: string; name: string; type: DosingProductType; unit: string; notes?: string
}) {
  return apiFetch<{ product: DosingProduct }>(`/api/dosing/products/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteDosingProduct(id: number) {
  return apiFetch<{ ok: boolean }>(`/api/dosing/products/${id}`, { method: 'DELETE' })
}

export function getDosingSchedules() {
  return apiFetch<{ schedules: DosingSchedule[] }>('/api/dosing/schedules')
}

export function createDosingSchedule(data: {
  product_id: number; amount: number; frequency: DosingFrequency;
  interval_days?: number; day_of_week?: number; enabled?: boolean;
  follow_up_parameter_id?: number; follow_up_days?: number; notes?: string
}) {
  return apiFetch<{ schedule: DosingSchedule }>('/api/dosing/schedules', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateDosingSchedule(id: number, data: {
  product_id: number; amount: number; frequency: DosingFrequency;
  interval_days?: number; day_of_week?: number; enabled: boolean;
  follow_up_parameter_id?: number; follow_up_days?: number; notes?: string
}) {
  return apiFetch<{ schedule: DosingSchedule }>(`/api/dosing/schedules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteDosingSchedule(id: number) {
  return apiFetch<{ ok: boolean }>(`/api/dosing/schedules/${id}`, { method: 'DELETE' })
}

export function logDose(scheduleId: number, data: {
  amount?: number; dosed_at?: string; notes?: string
}) {
  return apiFetch<{ log_id: number }>(`/api/dosing/schedules/${scheduleId}/log`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getDosingLogs(params?: { product_id?: number; limit?: number }) {
  const search = new URLSearchParams()
  if (params?.product_id) search.set('product_id', String(params.product_id))
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ logs: DosingLog[] }>(`/api/dosing/logs${qs ? `?${qs}` : ''}`)
}

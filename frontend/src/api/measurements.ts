import { apiFetch } from './_fetch'
import type { MeasurementParameter, Measurement, KitDef } from './types'

export function getMeasurementParameters() {
  return apiFetch<{ parameters: MeasurementParameter[] }>('/api/measurements/parameters')
}

export function getMeasurements(params?: {
  parameter?: string
  from?: string
  to?: string
  limit?: number
}) {
  const search = new URLSearchParams()
  if (params?.parameter) search.set('parameter', params.parameter)
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ measurements: Measurement[] }>(`/api/measurements${qs ? `?${qs}` : ''}`)
}

export function createMeasurement(data: {
  parameter: string
  value: number
  measured_at: string
  notes?: string | null
  test_kit_ref?: string | null
  raw_value?: number | null
}) {
  return apiFetch<Measurement>('/api/measurements', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateMeasurement(
  id: number,
  data: {
    parameter?: string
    value?: number
    measured_at?: string
    notes?: string | null
    test_kit_ref?: string | null
    raw_value?: number | null
  },
) {
  return apiFetch<Measurement>(`/api/measurements/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteMeasurement(id: number) {
  return apiFetch<void>(`/api/measurements/${id}`, { method: 'DELETE' })
}

export function getKitCatalog() {
  return apiFetch<{ kits: Record<string, KitDef[]> }>('/api/measurements/kits')
}

const KIT_PREF_PREFIX = 'kit_pref:'

export function getKitPref(paramName: string): string | null {
  try { return localStorage.getItem(KIT_PREF_PREFIX + paramName) } catch { return null }
}

export function setKitPref(paramName: string, ref: string) {
  try { localStorage.setItem(KIT_PREF_PREFIX + paramName, ref) } catch {}
}

export function clearKitPref(paramName: string) {
  try { localStorage.removeItem(KIT_PREF_PREFIX + paramName) } catch {}
}

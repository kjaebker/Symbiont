import { apiFetch, uploadMultipart } from './_fetch'
import type { LivestockItem, LivestockObservation, LivestockType, LivestockStatus } from './types'

export function getLivestock(params?: { type?: LivestockType; status?: LivestockStatus }) {
  const search = new URLSearchParams()
  if (params?.type) search.set('type', params.type)
  if (params?.status) search.set('status', params.status)
  const qs = search.toString()
  return apiFetch<{ livestock: LivestockItem[] }>(`/api/livestock${qs ? `?${qs}` : ''}`)
}

export function getLivestockSpecies() {
  return apiFetch<{ species: string[] }>('/api/livestock/species')
}

export function getLivestockItem(id: number) {
  return apiFetch<LivestockItem>(`/api/livestock/${id}`)
}

export function createLivestockItem(
  data: Omit<LivestockItem, 'id' | 'created_at' | 'updated_at' | 'image_path'>,
) {
  return apiFetch<LivestockItem>('/api/livestock', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateLivestockItem(
  id: number,
  data: Partial<Omit<LivestockItem, 'id' | 'created_at' | 'updated_at'>>,
) {
  return apiFetch<LivestockItem>(`/api/livestock/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteLivestockItem(id: number) {
  return apiFetch<void>(`/api/livestock/${id}`, { method: 'DELETE' })
}

export function uploadLivestockImage(id: number, file: File) {
  return uploadMultipart<{ image_path: string }>(`/api/livestock/${id}/image`, file)
}

export function editLivestockImage(id: number, file: File) {
  return uploadMultipart<{ image_path: string }>(`/api/livestock/${id}/image/edit`, file)
}

export function resetLivestockImage(id: number) {
  return apiFetch<{ image_path: string }>(`/api/livestock/${id}/image/reset`, { method: 'POST' })
}

export function deleteLivestockImage(id: number) {
  return apiFetch<{ status: string }>(`/api/livestock/${id}/image`, { method: 'DELETE' })
}

export function getLivestockObservations(livestockId: number) {
  return apiFetch<{ observations: LivestockObservation[] }>(
    `/api/livestock/${livestockId}/observations`,
  )
}

export function createLivestockObservation(
  livestockId: number,
  data: { status?: LivestockStatus | null; note?: string | null },
) {
  return apiFetch<LivestockObservation>(`/api/livestock/${livestockId}/observations`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function uploadObservationImage(livestockId: number, obsId: number, file: File) {
  return uploadMultipart<{ image_path: string }>(
    `/api/livestock/${livestockId}/observations/${obsId}/image`,
    file,
  )
}

export function editObservationImage(livestockId: number, obsId: number, file: File) {
  return uploadMultipart<{ image_path: string }>(
    `/api/livestock/${livestockId}/observations/${obsId}/image/edit`,
    file,
  )
}

export function resetObservationImage(livestockId: number, obsId: number) {
  return apiFetch<{ image_path: string }>(
    `/api/livestock/${livestockId}/observations/${obsId}/image/reset`,
    { method: 'POST' },
  )
}

export function deleteObservationImage(livestockId: number, obsId: number) {
  return apiFetch<{ status: string }>(
    `/api/livestock/${livestockId}/observations/${obsId}/image`,
    { method: 'DELETE' },
  )
}

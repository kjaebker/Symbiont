import { apiFetch, uploadMultipart } from './_fetch'
import type { Device, DeviceOutlet, DeviceSuggestion } from './types'

export function getDevices() {
  return apiFetch<{ devices: Device[] }>('/api/devices')
}

export function getDevice(id: number) {
  return apiFetch<Device>(`/api/devices/${id}`)
}

export function createDevice(device: Omit<Device, 'id' | 'created_at' | 'updated_at' | 'image_path'>) {
  return apiFetch<Device>('/api/devices', {
    method: 'POST',
    body: JSON.stringify(device),
  })
}

export function updateDevice(id: number, device: Partial<Device>) {
  return apiFetch<Device>(`/api/devices/${id}`, {
    method: 'PUT',
    body: JSON.stringify(device),
  })
}

export function deleteDevice(id: number) {
  return apiFetch<void>(`/api/devices/${id}`, { method: 'DELETE' })
}

export function setDeviceProbes(id: number, probeNames: string[]) {
  return apiFetch<Device>(`/api/devices/${id}/probes`, {
    method: 'PUT',
    body: JSON.stringify({ probe_names: probeNames }),
  })
}

export function setDeviceOutlets(id: number, outlets: DeviceOutlet[]) {
  return apiFetch<Device>(`/api/devices/${id}/outlets`, {
    method: 'PUT',
    body: JSON.stringify({ outlet_ids: outlets }),
  })
}

export function uploadDeviceImage(id: number, file: File) {
  return uploadMultipart<{ image_path: string }>(`/api/devices/${id}/image`, file)
}

export function deleteDeviceImage(id: number) {
  return apiFetch<void>(`/api/devices/${id}/image`, { method: 'DELETE' })
}

export function getDeviceSuggestions() {
  return apiFetch<{ suggestions: DeviceSuggestion[] }>('/api/devices/suggestions')
}

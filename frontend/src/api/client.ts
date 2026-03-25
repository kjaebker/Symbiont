import type {
  Probe,
  ProbeHistory,
  Outlet,
  OutletEvent,
  SystemStatus,
  AlertRule,
  AlertEvent,
  ProbeConfig,
  OutletConfig,
  AuthToken,
  BackupJob,
  NotificationTarget,
  NotificationTestResult,
  SystemLogLine,
  Device,
  DeviceSuggestion,
  DashboardItem,
} from './types'

const TOKEN_KEY = 'symbiont_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

class APIRequestError extends Error {
  code: string
  status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.name = 'APIRequestError'
    this.code = code
    this.status = status
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    ...((init?.headers as Record<string, string>) ?? {}),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  if (init?.body && typeof init.body === 'string') {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(path, { ...init, headers })

  if (res.status === 401) {
    clearToken()
    window.location.href = '/login'
    throw new APIRequestError('Unauthorized', 'unauthorized', 401)
  }

  if (res.status === 204) {
    return undefined as T
  }

  const body = await res.json()

  if (!res.ok) {
    throw new APIRequestError(
      body.error ?? 'Unknown error',
      body.code ?? 'unknown',
      res.status,
    )
  }

  return body as T
}

// Probes
export function getProbes() {
  return apiFetch<{ probes: Probe[]; polled_at: string }>('/api/probes')
}

export function getProbeHistory(
  name: string,
  params?: { from?: string; to?: string; interval?: string },
) {
  const search = new URLSearchParams()
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  if (params?.interval) search.set('interval', params.interval)
  const qs = search.toString()
  return apiFetch<ProbeHistory>(`/api/probes/${encodeURIComponent(name)}/history${qs ? `?${qs}` : ''}`)
}

// Outlets
export function getOutlets() {
  return apiFetch<{ outlets: Outlet[] }>('/api/outlets')
}

export function setOutletState(id: string, state: 'ON' | 'OFF' | 'AUTO') {
  return apiFetch<{ id: string; name: string; state: string; logged_at: string }>(
    `/api/outlets/${encodeURIComponent(id)}`,
    { method: 'PUT', body: JSON.stringify({ state }) },
  )
}

export function getOutletEvents(params?: { outlet_id?: string; initiated_by?: string; limit?: number }) {
  const search = new URLSearchParams()
  if (params?.outlet_id) search.set('outlet_id', params.outlet_id)
  if (params?.initiated_by) search.set('initiated_by', params.initiated_by)
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ events: OutletEvent[] }>(`/api/outlets/events${qs ? `?${qs}` : ''}`)
}

// System
export function getSystemStatus() {
  return apiFetch<SystemStatus>('/api/system')
}

// Alerts
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

// Config
export function getProbeConfigs() {
  return apiFetch<{ configs: ProbeConfig[] }>('/api/config/probes')
}

export function updateProbeConfig(name: string, config: Partial<ProbeConfig>) {
  return apiFetch<ProbeConfig>(`/api/config/probes/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

export function getOutletConfigs() {
  return apiFetch<{ configs: OutletConfig[] }>('/api/config/outlets')
}

export function updateOutletConfig(id: string, config: Partial<OutletConfig>) {
  return apiFetch<OutletConfig>(`/api/config/outlets/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

// Tokens
export function listTokens() {
  return apiFetch<{ tokens: AuthToken[] }>('/api/tokens')
}

export function createToken(label: string) {
  return apiFetch<{ token: string; label: string }>('/api/tokens', {
    method: 'POST',
    body: JSON.stringify({ label }),
  })
}

export function revokeToken(id: number) {
  return apiFetch<void>(`/api/tokens/${id}`, { method: 'DELETE' })
}

// Alert Events
export function getAlertEvents(params?: { rule_id?: number; active_only?: boolean; limit?: number }) {
  const search = new URLSearchParams()
  if (params?.rule_id) search.set('rule_id', String(params.rule_id))
  if (params?.active_only) search.set('active_only', 'true')
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ events: AlertEvent[] }>(`/api/alerts/events${qs ? `?${qs}` : ''}`)
}

// Notification Targets
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

// Backup
export function getBackups() {
  return apiFetch<{ backups: BackupJob[] }>('/api/system/backups')
}

export function triggerBackup() {
  return apiFetch<BackupJob>('/api/system/backups', { method: 'POST' })
}

// Devices
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

export function uploadDeviceImage(id: number, file: File) {
  const form = new FormData()
  form.append('image', file)
  const token = getToken()
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`
  return fetch(`/api/devices/${id}/image`, {
    method: 'POST',
    headers,
    body: form,
  }).then(async (res) => {
    if (!res.ok) {
      const body = await res.json()
      throw new Error(body.error ?? 'Upload failed')
    }
    return res.json() as Promise<{ image_path: string }>
  })
}

export function deleteDeviceImage(id: number) {
  return apiFetch<void>(`/api/devices/${id}/image`, { method: 'DELETE' })
}

export function getDeviceSuggestions() {
  return apiFetch<{ suggestions: DeviceSuggestion[] }>('/api/devices/suggestions')
}

// Dashboard layout
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

// System log
export function getSystemLog(params?: { limit?: number; service?: string }) {
  const search = new URLSearchParams()
  if (params?.limit) search.set('limit', String(params.limit))
  if (params?.service) search.set('service', params.service)
  const qs = search.toString()
  return apiFetch<{ lines: SystemLogLine[] }>(`/api/system/log${qs ? `?${qs}` : ''}`)
}

import type {
  Probe,
  ProbeHistory,
  Outlet,
  OutletEvent,
  SystemStatus,
  AlertRule,
  ProbeConfig,
  OutletConfig,
  AuthToken,
  BackupJob,
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

export function getOutletEvents(params?: { outlet_id?: string; limit?: number }) {
  const search = new URLSearchParams()
  if (params?.outlet_id) search.set('outlet_id', params.outlet_id)
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

// Backup
export function getBackups() {
  return apiFetch<{ backups: BackupJob[] }>('/api/backups')
}

export function triggerBackup() {
  return apiFetch<BackupJob>('/api/backups', { method: 'POST' })
}

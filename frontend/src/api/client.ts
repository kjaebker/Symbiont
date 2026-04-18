import type {
  Probe,
  ProbeHistory,
  Outlet,
  SystemStatus,
  FeedStatus,
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
  MeasurementParameter,
  KitDef,
  Measurement,
  LivestockItem,
  LivestockObservation,
  LivestockType,
  LivestockStatus,
  AgentSettings,
  AgentSkill,
  DailyPrompt,
} from './types'

export type { Outlet, DailyPrompt }

const TOKEN_KEY = 'symbiont_token'
const BUBBLES_KEY = 'ui:bubbles'

/**
 * Derives the thumbnail URL from an image_path using the same naming convention
 * as the API server: replace the file extension with "-thumb.jpg".
 *
 * E.g. "images/livestock-1-123.jpg" → "/images/livestock-1-123-thumb.jpg"
 */
export function thumbUrl(imagePath: string): string {
  const stem = imagePath.replace(/\.[^./]+$/, '')
  return `/${stem}-thumb.jpg`
}

export function getBubblesEnabled(): boolean {
  try { return localStorage.getItem(BUBBLES_KEY) !== 'false' } catch { return true }
}

export function setBubblesEnabled(value: boolean): void {
  try { localStorage.setItem(BUBBLES_KEY, String(value)) } catch {}
}

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


// Feed mode
export function getFeedStatus() {
  return apiFetch<FeedStatus>('/api/feed')
}

export function setFeedMode(name: number, active: boolean) {
  return apiFetch<FeedStatus>('/api/feed', {
    method: 'PUT',
    body: JSON.stringify({ name, active }),
  })
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

export function createToken(label: string, scope: string = 'admin') {
  return apiFetch<{ token: string; label: string; scope: string }>('/api/tokens', {
    method: 'POST',
    body: JSON.stringify({ label, scope }),
  })
}

export function updateTokenScope(id: number, scope: string) {
  return apiFetch<{ id: string; scope: string }>(`/api/tokens/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ scope }),
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
  return apiFetch<BackupJob>('/api/system/backup', { method: 'POST' })
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

// Measurements
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

// Livestock
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
  const token = getToken()
  const form = new FormData()
  form.append('image', file)
  return fetch(`/api/livestock/${id}/image`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  }).then(async (res) => {
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: 'upload failed', code: 'upload_error' }))
      throw new Error(err.error ?? 'upload failed')
    }
    return res.json() as Promise<{ image_path: string }>
  })
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
  const token = getToken()
  const form = new FormData()
  form.append('image', file)
  return fetch(`/api/livestock/${livestockId}/observations/${obsId}/image`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  }).then(async (res) => {
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: 'upload failed', code: 'upload_error' }))
      throw new Error(err.error ?? 'upload failed')
    }
    return res.json() as Promise<{ image_path: string }>
  })
}

export function deleteObservationImage(livestockId: number, obsId: number) {
  return apiFetch<{ status: string }>(
    `/api/livestock/${livestockId}/observations/${obsId}/image`,
    { method: 'DELETE' },
  )
}

// ─── Tank Profile ────────────────────────────────────────────────────────────

export type TankSection = 'display' | 'sump'
export type TankType = 'reef' | 'fowlr' | 'mixed' | 'nano' | 'freshwater' | 'other'

export interface TankProfile {
  section: TankSection
  display_name: string | null
  volume_gallons: number | null
  length_in: number | null
  width_in: number | null
  height_in: number | null
  tank_type: TankType | null
  manufacturer: string | null
  model: string | null
  setup_date: string | null // YYYY-MM-DD
  notes: string | null
  updated_at: string
}

export type TankProfileInput = Omit<TankProfile, 'section' | 'updated_at'>

export function getTankProfile() {
  return apiFetch<{ display: TankProfile | null; sump: TankProfile | null }>('/api/tank/profile')
}

export function upsertTankProfile(section: TankSection, data: Partial<TankProfileInput>) {
  return apiFetch<TankProfile>(`/api/tank/profile/${section}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

// ─── Journal ─────────────────────────────────────────────────────────────────

export type JournalCategory = 'observation' | 'maintenance' | 'event' | 'milestone'
export type JournalSentiment = 'good' | 'neutral' | 'bad' | 'critical'
export type JournalSource = 'manual' | 'system' | 'ai'

export interface JournalEntry {
  id: number
  ts: string
  category: JournalCategory
  sentiment: JournalSentiment | null
  title: string
  body: string | null
  source: JournalSource
  source_ref: string | null
  created_at: string
}

export interface JournalTemplate {
  category: JournalCategory
  sentiment: JournalSentiment
  title: string
}

export interface JournalListParams {
  category?: JournalCategory
  sentiment?: JournalSentiment
  from?: string
  to?: string
  limit?: number
}

export function getJournalTemplates() {
  return apiFetch<{ templates: Record<JournalCategory, JournalTemplate[]> }>('/api/journal/templates')
}

export function listJournalEntries(params?: JournalListParams) {
  const search = new URLSearchParams()
  if (params?.category) search.set('category', params.category)
  if (params?.sentiment) search.set('sentiment', params.sentiment)
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ entries: JournalEntry[] }>(`/api/journal${qs ? '?' + qs : ''}`)
}

export function createJournalEntry(data: {
  category: JournalCategory
  sentiment?: JournalSentiment
  title: string
  body?: string
}) {
  return apiFetch<JournalEntry>('/api/journal', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateJournalEntry(id: number, data: {
  category: JournalCategory
  sentiment?: JournalSentiment | null
  title: string
  body?: string | null
}) {
  return apiFetch<JournalEntry>(`/api/journal/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteJournalEntry(id: number) {
  return apiFetch<{ status: string }>(`/api/journal/${id}`, { method: 'DELETE' })
}

// ─── Daily Prompt ─────────────────────────────────────────────────────────────

export function getDailyPrompt() {
  return apiFetch<{ prompt: DailyPrompt | null }>('/api/daily-prompt')
}

export function respondToPrompt(question: string, response: string) {
  return apiFetch<{ ok: boolean }>('/api/daily-prompt/respond', {
    method: 'POST',
    body: JSON.stringify({ question, response }),
  })
}

// ─── Audit Events ─────────────────────────────────────────────────────────────

export interface AuditEvent {
  id: number
  ts: string
  kind: string
  payload_json: string
  correlation_id?: string
}

export interface SubscriberStat {
  name: string
  queue_depth: number
  published: number
  dropped: number
  last_error?: string
}

export interface AuditEventListParams {
  kind?: string
  since?: string // RFC3339
  limit?: number
  correlation_id?: string
  initiated_by?: string
}

export function listAuditEvents(params?: AuditEventListParams) {
  const q = new URLSearchParams()
  if (params?.kind) q.set('kind', params.kind)
  if (params?.since) q.set('since', params.since)
  if (params?.limit) q.set('limit', String(params.limit))
  if (params?.correlation_id) q.set('correlation_id', params.correlation_id)
  if (params?.initiated_by) q.set('initiated_by', params.initiated_by)
  const qs = q.toString()
  return apiFetch<{ events: AuditEvent[] }>(`/api/events${qs ? `?${qs}` : ''}`)
}

export function getEventBusStats() {
  return apiFetch<{ subscribers: SubscriberStat[] }>('/api/events/stats')
}

// Agent
export function getAgentSettings() {
  return apiFetch<AgentSettings>('/api/agent/settings')
}

export function updateAgentSettings(patch: Partial<AgentSettings>) {
  return apiFetch<AgentSettings>('/api/agent/settings', {
    method: 'PUT',
    body: JSON.stringify(patch),
  })
}

export function getAgentContext() {
  return apiFetch<{ context: string }>('/api/agent/context')
}

export function getAgentSkills() {
  return apiFetch<{ skills: AgentSkill[] }>('/api/agent/skills')
}

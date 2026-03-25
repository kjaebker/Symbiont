export type ProbeStatus = 'normal' | 'warning' | 'critical' | 'unknown'

export interface Probe {
  name: string
  display_name: string
  type: string
  value: number
  unit: string
  ts: string
  status: ProbeStatus
  display_order: number
  hidden: boolean
}

export interface ProbeHistoryPoint {
  ts: string
  value: number
}

export interface ProbeHistory {
  probe: string
  from: string
  to: string
  interval: string
  data: ProbeHistoryPoint[]
}

export type OutletState = 'ON' | 'OFF' | 'AON' | 'AOF' | 'TBL' | 'PF1' | 'PF2' | 'PF3' | 'PF4'

export interface Outlet {
  id: string
  name: string
  display_name: string
  state: OutletState
  type: string
  intensity: number
  display_order: number
  hidden: boolean
}

export interface OutletEvent {
  id: number
  ts: string
  outlet_id: string
  outlet_name: string
  outlet_display_name?: string
  from_state: string
  to_state: string
  initiated_by: string
}

export interface AlertRule {
  id: number
  probe_name: string
  probe_display_name?: string
  condition: 'above' | 'below' | 'outside_range'
  threshold_low: number | null
  threshold_high: number | null
  severity: 'warning' | 'critical'
  cooldown_minutes: number
  enabled: boolean
  created_at: string
}

export interface SystemStatus {
  controller: {
    serial: string
    firmware: string
    hardware: string
  }
  poller: {
    last_poll_ts: string
    poll_ok: boolean
    poll_interval_seconds: number
  }
  db: {
    duckdb_size_bytes: number
    sqlite_size_bytes: number
  }
}

export interface ProbeConfig {
  probe_name: string
  display_name: string
  unit_override: string
  display_order: number
  min_normal: number | null
  max_normal: number | null
  min_warning: number | null
  max_warning: number | null
  hidden: boolean
}

export interface OutletConfig {
  outlet_id: string
  display_name: string
  display_order: number
  icon: string
  hidden: boolean
}

export interface BackupJob {
  id: number
  ts: string
  status: 'success' | 'failed'
  path: string
  size_bytes: number
  error: string
}

export interface AuthToken {
  id: number
  label: string
  created_at: string
  last_used: string | null
}

export interface AlertEvent {
  id: number
  rule_id: number
  fired_at: string
  cleared_at: string | null
  peak_value: number
  notified: boolean
  probe_name: string | null
  probe_display_name?: string
  severity: string | null
}

export interface NotificationTarget {
  id: number
  type: string
  config: string
  label: string
  enabled: boolean
}

export interface NotificationTestResult {
  label: string
  success: boolean
  error?: string
}

export interface SystemLogLine {
  ts: string
  service: string
  level: string
  msg: string
  fields?: Record<string, unknown>
}

export interface APIError {
  error: string
  code: string
}

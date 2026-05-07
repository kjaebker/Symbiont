import { apiFetch } from './_fetch'

export interface TDataPoint {
  t: number    // seconds since midnight
  ch: number[] // 13 channel values (0–100)
}

export interface OutputProgram {
  did: string
  name: string
  type: string
  icon: string
  prog: string
  tdata?: TDataPoint[]
}

export function getPrograms() {
  return apiFetch<{ programs: OutputProgram[]; synced_at: string | null }>('/api/programs')
}

export function syncPrograms() {
  return apiFetch<{ total: number; changed: number }>('/api/programs/sync', { method: 'POST' })
}

import { apiFetch } from './_fetch'

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

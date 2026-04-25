export const typePersonality = {
  heater:    { color: '#ff8796', bg: 'rgba(255,135,150,0.13)', blob: '60% 40% 30% 70% / 60% 30% 70% 40%', label: 'HEATER' },
  chemistry: { color: '#6dfe9c', bg: 'rgba(109,254,156,0.11)', blob: '40% 60% 70% 30% / 50% 60% 40% 50%', label: 'CHEMISTRY' },
  pump:      { color: '#3adffa', bg: 'rgba(58,223,250,0.12)',  blob: '70% 30% 40% 60% / 40% 70% 30% 60%', label: 'PUMP' },
  ato:       { color: '#3adffa', bg: 'rgba(58,223,250,0.10)',  blob: '50% 50% 40% 60% / 60% 40% 50% 50%', label: 'ATO' },
  skimmer:   { color: '#b09fff', bg: 'rgba(176,159,255,0.11)', blob: '30% 70% 60% 40% / 40% 60% 70% 30%', label: 'SKIMMER' },
  wavemaker: { color: '#3adffa', bg: 'rgba(58,223,250,0.10)',  blob: '60% 40% 50% 50% / 50% 60% 40% 50%', label: 'WAVEMAKER' },
  light:     { color: '#ffd264', bg: 'rgba(255,210,100,0.10)', blob: '50% 60% 40% 60% / 60% 50% 60% 40%', label: 'LIGHT' },
  reactor:   { color: '#6dfe9c', bg: 'rgba(109,254,156,0.11)', blob: '40% 60% 70% 30% / 50% 60% 40% 50%', label: 'REACTOR' },
  chiller:   { color: '#3adffa', bg: 'rgba(58,223,250,0.10)',  blob: '50% 50% 60% 40% / 40% 60% 50% 50%', label: 'CHILLER' },
  fan:       { color: '#3adffa', bg: 'rgba(58,223,250,0.10)',  blob: '60% 40% 60% 40% / 40% 60% 40% 60%', label: 'FAN' },
  doser:     { color: '#6dfe9c', bg: 'rgba(109,254,156,0.11)', blob: '45% 55% 65% 35% / 55% 45% 55% 45%', label: 'DOSER' },
} as const

export const defaultPersonality = {
  color: '#3adffa',
  bg: 'rgba(58,223,250,0.10)',
  blob: '50% 50% 50% 50%',
  label: 'DEVICE',
}

export type PersonalityKey = keyof typeof typePersonality

export function getPersonality(type: string) {
  return (typePersonality as Record<string, typeof defaultPersonality>)[type] ?? defaultPersonality
}

/** Infer personality from device_type (if set) then fall back to name-based matching. */
export function inferPersonality(name: string, deviceType: string | null | undefined) {
  if (deviceType) {
    const byType = (typePersonality as Record<string, typeof defaultPersonality>)[deviceType.toLowerCase()]
    if (byType) return byType
  }
  const n = (name ?? '').toLowerCase()
  if (/heat/.test(n)) return typePersonality.heater
  if (/skim/.test(n)) return typePersonality.skimmer
  if (/gyre|wave|tunze|vortech/.test(n)) return typePersonality.wavemaker
  if (/light|led|kessil|radion|hydra|blu\b|wht\b|grow/.test(n)) return typePersonality.light
  if (/\bato\b|auto.?top|top.?off|ato\s/.test(n)) return typePersonality.ato
  if (/chill/.test(n)) return typePersonality.chiller
  if (/\bfan\b/.test(n)) return typePersonality.fan
  if (/reactor|kalk|carx/.test(n)) return typePersonality.reactor
  if (/doser|\bdos\b|two.?part/.test(n)) return typePersonality.doser
  if (/chem|calcium|magnesium|alkalin/.test(n)) return typePersonality.chemistry
  if (/pump|return|sump|flow/.test(n)) return typePersonality.pump
  return defaultPersonality
}

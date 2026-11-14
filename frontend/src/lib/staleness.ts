import { daysSince } from './utils'

/**
 * How old a reading can get before the UI stops presenting it as current.
 * Past these ages a value is dimmed and its age is shown in the warning color.
 */
export const STALE_AFTER_DAYS = {
  /** Manual water tests (alkalinity, calcium, …). */
  measurement: 30,
  /** A sick or quarantined animal with no observation logged. */
  livestockObservation: 14,
} as const

export function isStale(ts: string | null | undefined, kind: keyof typeof STALE_AFTER_DAYS): boolean {
  if (!ts) return false
  return daysSince(ts) > STALE_AFTER_DAYS[kind]
}

import { overdueBy } from '@/lib/utils'
import type { DosingFrequency } from '@/api/client'

export const DOSING_FREQUENCY_LABELS: Record<DosingFrequency, string> = {
  daily: 'Daily',
  twice_daily: 'Twice daily',
  every_n_days: 'Every N days',
  weekly: 'Weekly',
  as_needed: 'As needed',
}

/** "Overdue 3d", "Due today", or a short date for a dose's next due time. */
export function formatDoseDue(iso: string | null): string {
  if (!iso) return 'Not scheduled'
  const d = new Date(iso)
  const diff = d.getTime() - Date.now()
  if (diff < 0) {
    const late = overdueBy(iso)
    return late ? `Overdue ${late}` : 'Overdue'
  }
  if (diff < 3600_000) return 'Due soon'
  if (diff < 86400_000) return 'Due today'
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

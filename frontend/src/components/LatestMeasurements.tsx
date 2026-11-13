import { useMeasurements } from '@/hooks/useMeasurements'
import { isStale } from '@/lib/staleness'
import { cn } from '@/lib/utils'
import { TimeAgo, labelClass } from '@/components/ui'
import type { MeasurementParameter } from '@/api/types'

function formatValue(v: number): string {
  return v % 1 === 0 ? v.toFixed(0) : v.toFixed(2)
}

interface TileProps {
  parameter: string
  selected: boolean
  onSelect: () => void
}

function LatestTile({ parameter, selected, onSelect }: TileProps) {
  const { data, isLoading } = useMeasurements({ parameter, limit: 1 })
  const latest = data?.measurements[0] ?? null
  const stale = isStale(latest?.measured_at, 'measurement')

  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      className={cn(
        'rounded-2xl px-4 py-3 text-left flex flex-col items-start justify-start cursor-pointer transition-fluid',
        selected ? 'bg-primary/15' : 'bg-surface-container hover:bg-surface-container-high',
      )}
    >
      <div className={cn(labelClass, 'truncate')}>{parameter}</div>
      {isLoading ? (
        <div className="h-7 w-16 mt-1.5 rounded bg-surface-container-high animate-pulse" />
      ) : latest ? (
        <>
          <div className="flex items-baseline gap-1 mt-1">
            <span className={cn('text-2xl font-bold tracking-tight', stale ? 'text-on-surface-dim' : 'text-on-surface')}>
              {formatValue(latest.value)}
            </span>
            {latest.canonical_unit && (
              <span className="text-xs text-on-surface-dim">{latest.canonical_unit}</span>
            )}
          </div>
          <TimeAgo
            ts={latest.measured_at}
            className={cn('text-xs', stale ? 'text-warning' : 'text-on-surface-faint')}
          />
        </>
      ) : (
        <div className="mt-1.5 text-sm text-on-surface-faint">Not tested</div>
      )}
    </button>
  )
}

interface LatestMeasurementsProps {
  parameters: MeasurementParameter[]
  selected: string | null
  /** Called with the parameter to filter by, or null to clear the filter. */
  onSelect: (parameter: string | null) => void
}

/**
 * The most recent test result for every parameter, with its age. Results
 * older than the staleness limit are dimmed and their age turns amber, so an
 * old number is not mistaken for the tank's current chemistry.
 */
export function LatestMeasurements({ parameters, selected, onSelect }: LatestMeasurementsProps) {
  if (parameters.length === 0) return null
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
      {parameters.map((p) => (
        <LatestTile
          key={p.id}
          parameter={p.name}
          selected={selected === p.name}
          onSelect={() => onSelect(selected === p.name ? null : p.name)}
        />
      ))}
    </div>
  )
}

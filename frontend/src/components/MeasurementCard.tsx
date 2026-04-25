import { useNavigate } from 'react-router-dom'
import { FlaskConical } from 'lucide-react'
import { useMeasurements } from '@/hooks/useMeasurements'
import { relativeTime } from '@/lib/utils'

interface MeasurementCardProps {
  parameter: string
}

export function MeasurementCard({ parameter }: MeasurementCardProps) {
  const navigate = useNavigate()
  const { data, isLoading } = useMeasurements({ parameter, limit: 1 })
  const latest = data?.measurements[0] ?? null

  const displayValue = latest
    ? latest.value % 1 === 0
      ? latest.value.toFixed(0)
      : latest.value.toFixed(2)
    : null

  return (
    <button
      onClick={() => navigate('/measurements')}
      className="rounded-2xl p-5 text-left transition-fluid hover:shadow-glow-secondary cursor-pointer w-full"
      style={{ background: 'linear-gradient(145deg, var(--color-surface-container) 20%, rgba(109,254,156,0.22))' }}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div
            className="w-8 h-8 flex items-center justify-center shrink-0"
            style={{
              borderRadius: '40% 60% 70% 30% / 50% 60% 40% 50%',
              background: 'rgba(109, 254, 156, 0.10)',
            }}
          >
            <FlaskConical size={14} className="text-secondary" />
          </div>
          <span className="text-xs text-on-surface-dim uppercase tracking-widest font-medium">
            {parameter}
          </span>
        </div>
        <span className="text-xs text-on-surface-faint">
          {latest ? relativeTime(latest.measured_at) : ''}
        </span>
      </div>

      {isLoading ? (
        <div className="h-10 w-24 bg-surface-container-high rounded animate-pulse" />
      ) : latest ? (
        <div className="flex items-baseline gap-1.5">
          <span className="text-4xl font-bold text-secondary tracking-tight text-glow-secondary">
            {displayValue}
          </span>
          {latest.canonical_unit && (
            <span className="text-lg text-on-surface-dim font-light">
              {latest.canonical_unit}
            </span>
          )}
        </div>
      ) : (
        <div className="flex items-center gap-2 h-10">
          <span className="text-sm text-on-surface-faint">No data yet</span>
        </div>
      )}

      {/* Spacer to align height with ProbeCard */}
      <div className="h-[28px] mt-1" />

      <div className="h-10 flex items-end">
        {latest?.notes && (
          <p className="text-xs text-on-surface-faint truncate">{latest.notes}</p>
        )}
      </div>
    </button>
  )
}

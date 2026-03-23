import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Zap } from 'lucide-react'
import { getProbeHistory } from '@/api/client'
import type { Probe } from '@/api/types'
import { cn } from '@/lib/utils'
import { Sparkline } from './Sparkline'

const statusColor = {
  normal: 'bg-secondary',
  warning: 'bg-amber-400',
  critical: 'bg-tertiary',
  unknown: 'bg-on-surface-faint',
} as const

interface PowerPairCardProps {
  watts: Probe
  amps: Probe
  label: string
}

export function PowerPairCard({ watts, amps, label }: PowerPairCardProps) {
  const navigate = useNavigate()

  const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
  const { data: history } = useQuery({
    queryKey: ['probeSparkline', watts.name],
    queryFn: () =>
      getProbeHistory(watts.name, { from: twoHoursAgo, interval: '5m' }),
    staleTime: 60_000,
  })

  const sparklineData = history?.data.map((d) => d.value) ?? []

  // Use the worse status of the two
  const worstStatus =
    watts.status === 'critical' || amps.status === 'critical'
      ? 'critical'
      : watts.status === 'warning' || amps.status === 'warning'
        ? 'warning'
        : watts.status === 'normal' || amps.status === 'normal'
          ? 'normal'
          : 'unknown'

  return (
    <button
      onClick={() => navigate(`/history?probe=${encodeURIComponent(watts.name)}`)}
      className="bg-surface-container rounded-2xl p-5 text-left transition-fluid hover:bg-surface-container-high hover:shadow-glow-primary cursor-pointer w-full"
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Zap className="h-4 w-4 text-primary" />
          <span className="text-xs text-on-surface-dim uppercase tracking-widest font-medium">
            {label}
          </span>
        </div>
        <span
          className={cn(
            'h-2.5 w-2.5 rounded-full',
            statusColor[worstStatus],
            worstStatus === 'normal' && 'animate-bio-pulse',
          )}
        />
      </div>

      <div className="flex items-baseline gap-1.5 mb-1">
        <span className="text-4xl font-bold text-on-surface tracking-tight text-glow-primary">
          {watts.value.toFixed(1)}
        </span>
        <span className="text-lg text-on-surface-dim font-light">
          {watts.unit}
        </span>
      </div>

      <div className="flex items-baseline gap-1.5 mb-3">
        <span className="text-lg font-semibold text-on-surface-dim tracking-tight">
          {amps.value.toFixed(2)}
        </span>
        <span className="text-sm text-on-surface-faint font-light">
          {amps.unit}
        </span>
      </div>

      <div className="h-10">
        <Sparkline data={sparklineData} color="#3adffa" />
      </div>
    </button>
  )
}

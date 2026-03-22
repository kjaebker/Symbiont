import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
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

interface ProbeCardProps {
  probe: Probe
}

export function ProbeCard({ probe }: ProbeCardProps) {
  const navigate = useNavigate()

  const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
  const { data: history } = useQuery({
    queryKey: ['probeSparkline', probe.name],
    queryFn: () =>
      getProbeHistory(probe.name, { from: twoHoursAgo, interval: '5m' }),
    staleTime: 60_000,
  })

  const sparklineData = history?.data.map((d) => d.value) ?? []

  return (
    <button
      onClick={() => navigate(`/history?probe=${encodeURIComponent(probe.name)}`)}
      className="bg-surface-container rounded-2xl p-5 text-left transition-fluid hover:bg-surface-container-high hover:shadow-glow-primary cursor-pointer w-full"
    >
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs text-on-surface-dim uppercase tracking-widest font-medium">
          {probe.display_name}
        </span>
        <span
          className={cn(
            'h-2.5 w-2.5 rounded-full',
            statusColor[probe.status],
            probe.status === 'normal' && 'animate-bio-pulse',
          )}
        />
      </div>

      <div className="flex items-baseline gap-1.5 mb-3">
        <span className="text-4xl font-bold text-on-surface tracking-tight text-glow-primary">
          {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
        </span>
        <span className="text-lg text-on-surface-dim font-light">
          {probe.unit === 'F' ? '\u00B0F' : probe.unit}
        </span>
      </div>

      <div className="h-10">
        <Sparkline data={sparklineData} color="#3adffa" />
      </div>
    </button>
  )
}

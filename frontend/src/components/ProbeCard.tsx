import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Thermometer, FlaskConical, Zap, ToggleLeft } from 'lucide-react'
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

type ProbeCategory = 'temperature' | 'chemistry' | 'power' | 'digital'

const categoryConfig: Record<
  ProbeCategory,
  {
    icon: typeof Thermometer
    color: string
    sparklineColor: string
    glowClass: string
    hoverGlow: string
  }
> = {
  temperature: {
    icon: Thermometer,
    color: 'text-tertiary',
    sparklineColor: '#ff8796',
    glowClass: 'text-glow-tertiary',
    hoverGlow: 'hover:shadow-glow-tertiary',
  },
  chemistry: {
    icon: FlaskConical,
    color: 'text-secondary',
    sparklineColor: '#6dfe9c',
    glowClass: 'text-glow-secondary',
    hoverGlow: 'hover:shadow-glow-secondary',
  },
  power: {
    icon: Zap,
    color: 'text-primary',
    sparklineColor: '#3adffa',
    glowClass: 'text-glow-primary',
    hoverGlow: 'hover:shadow-glow-primary',
  },
  digital: {
    icon: ToggleLeft,
    color: 'text-on-surface-dim',
    sparklineColor: '#8a90a8',
    glowClass: '',
    hoverGlow: 'hover:shadow-glow-primary',
  },
}

function getCategory(type: string): ProbeCategory {
  switch (type) {
    case 'Temp':
      return 'temperature'
    case 'pH':
    case 'ORP':
      return 'chemistry'
    case 'Amps':
    case 'pwr':
    case 'volts':
      return 'power'
    case 'digital':
      return 'digital'
    default:
      return 'power'
  }
}

interface ProbeCardProps {
  probe: Probe
}

export function ProbeCard({ probe }: ProbeCardProps) {
  const navigate = useNavigate()
  const category = getCategory(probe.type)
  const config = categoryConfig[category]
  const Icon = config.icon

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
      className={cn(
        'bg-surface-container rounded-2xl p-5 text-left transition-fluid hover:bg-surface-container-high cursor-pointer w-full',
        config.hoverGlow,
      )}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Icon className={cn('h-4 w-4', config.color)} />
          <span className="text-xs text-on-surface-dim uppercase tracking-widest font-medium">
            {probe.display_name}
          </span>
        </div>
        <span
          className={cn(
            'h-2.5 w-2.5 rounded-full',
            statusColor[probe.status],
            probe.status === 'normal' && 'animate-bio-pulse',
          )}
        />
      </div>

      <div className="flex items-baseline gap-1.5 mb-1">
        <span className={cn('text-4xl font-bold text-on-surface tracking-tight', config.glowClass)}>
          {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
        </span>
        <span className="text-lg text-on-surface-dim font-light">
          {probe.unit}
        </span>
      </div>

      {/* Spacer to match PowerPairCard's secondary metric line */}
      <div className="h-[28px] mb-3" />

      <div className="h-10">
        <Sparkline data={sparklineData} color={config.sparklineColor} />
      </div>
    </button>
  )
}

export { getCategory, type ProbeCategory }

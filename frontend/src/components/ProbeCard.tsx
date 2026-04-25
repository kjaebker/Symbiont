import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Thermometer, FlaskConical, Zap, ToggleLeft, Droplets, AlertTriangle } from 'lucide-react'
import { getProbeHistory } from '@/api/client'
import type { Probe } from '@/api/types'
import { cn } from '@/lib/utils'
import { Sparkline } from './Sparkline'

type ProbeCategory = 'temperature' | 'chemistry' | 'power' | 'digital'

const categoryConfig: Record<
  ProbeCategory,
  {
    icon: typeof Thermometer
    color: string
    hexColor: string
    sparklineColor: string
    glowClass: string
    hoverGlow: string
    tint: string
    blobShape: string
    blobBg: string
  }
> = {
  temperature: {
    icon: Thermometer,
    color: 'text-tertiary',
    hexColor: '#ff8796',
    sparklineColor: '#ff8796',
    glowClass: 'text-glow-tertiary',
    hoverGlow: 'hover:shadow-glow-tertiary',
    tint: 'rgba(255, 135, 150, 0.18)',
    blobShape: '60% 40% 30% 70% / 60% 30% 70% 40%',
    blobBg: 'rgba(255, 135, 150, 0.12)',
  },
  chemistry: {
    icon: FlaskConical,
    color: 'text-secondary',
    hexColor: '#6dfe9c',
    sparklineColor: '#6dfe9c',
    glowClass: 'text-glow-secondary',
    hoverGlow: 'hover:shadow-glow-secondary',
    tint: 'rgba(109, 254, 156, 0.18)',
    blobShape: '40% 60% 70% 30% / 50% 60% 40% 50%',
    blobBg: 'rgba(109, 254, 156, 0.10)',
  },
  power: {
    icon: Zap,
    color: 'text-primary',
    hexColor: '#3adffa',
    sparklineColor: '#3adffa',
    glowClass: 'text-glow-primary',
    hoverGlow: 'hover:shadow-glow-primary',
    tint: 'rgba(58, 223, 250, 0.18)',
    blobShape: '70% 30% 40% 60% / 40% 70% 30% 60%',
    blobBg: 'rgba(58, 223, 250, 0.10)',
  },
  digital: {
    icon: ToggleLeft,
    color: 'text-on-surface-dim',
    hexColor: '#8a90a8',
    sparklineColor: '#8a90a8',
    glowClass: '',
    hoverGlow: 'hover:shadow-glow-primary',
    tint: 'transparent',
    blobShape: '50% 50% 60% 40% / 40% 60% 50% 50%',
    blobBg: 'rgba(255, 255, 255, 0.04)',
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

function getBinaryLabel(probe: Probe): string {
  if (probe.value !== 0) {
    return probe.on_label ?? 'On'
  }
  return probe.off_label ?? 'Off'
}

function getBinaryIcon(probe: Probe) {
  switch (probe.input_category) {
    case 'fluid': return Droplets
    case 'alarm': return AlertTriangle
    default: return ToggleLeft
  }
}

interface ProbeCardProps {
  probe: Probe
}

export function ProbeCard({ probe }: ProbeCardProps) {
  const navigate = useNavigate()
  const category = getCategory(probe.type)
  const config = categoryConfig[category]

  const isBinary = probe.is_binary
  const Icon = isBinary ? getBinaryIcon(probe) : config.icon

  const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
  const { data: history } = useQuery({
    queryKey: ['probeSparkline', probe.name],
    queryFn: () =>
      getProbeHistory(probe.name, { from: twoHoursAgo, interval: '5m' }),
    staleTime: 60_000,
    enabled: !isBinary,
  })

  const sparklineData = history?.data.map((d) => d.value) ?? []

  const isAlarmActive = probe.input_category === 'alarm' && probe.status === 'critical'
  const isFluidWarn = probe.input_category === 'fluid' && probe.status === 'warning'

  return (
    <button
      onClick={() =>
        !isBinary &&
        navigate(`/history?tab=telemetry&probe=${encodeURIComponent(probe.name)}`)
      }
      className={cn(
        'rounded-2xl p-5 text-left transition-fluid w-full relative overflow-hidden',
        !isBinary && 'cursor-pointer',
        isBinary && 'cursor-default',
        isAlarmActive ? 'hover:shadow-glow-tertiary' : config.hoverGlow,
      )}
      style={{
        background: isAlarmActive
          ? 'linear-gradient(145deg, var(--color-surface-container) 20%, rgba(255,135,150,0.22))'
          : `linear-gradient(145deg, var(--color-surface-container) 20%, ${config.tint})`,
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: -20,
          right: -20,
          width: 120,
          height: 120,
          borderRadius: config.blobShape,
          background: isAlarmActive ? 'rgba(255,135,150,0.18)' : config.blobBg,
          filter: 'blur(20px)',
          pointerEvents: 'none',
        }}
      />
      <div className="flex items-center gap-2 mb-3">
        <div
          className="w-8 h-8 flex items-center justify-center shrink-0"
          style={{
            borderRadius: isAlarmActive ? '60% 40% 30% 70% / 60% 30% 70% 40%' : config.blobShape,
            background: isAlarmActive ? 'rgba(255,135,150,0.22)'
              : (isFluidWarn || probe.status === 'warning') ? 'rgba(251,191,36,0.18)'
              : probe.status === 'critical' ? 'rgba(255,135,150,0.22)'
              : config.blobBg,
            boxShadow: isAlarmActive ? '0 0 16px rgba(255,135,150,0.55)'
              : (isFluidWarn || probe.status === 'warning') ? '0 0 14px rgba(251,191,36,0.45)'
              : probe.status === 'critical' ? '0 0 16px rgba(255,135,150,0.55)'
              : 'none',
            animation: isAlarmActive || probe.status === 'critical' ? 'bioluminescent-pulse 1.8s ease-in-out infinite'
              : isFluidWarn || probe.status === 'warning' ? 'bioluminescent-pulse 2.8s ease-in-out infinite'
              : undefined,
          }}
        >
          <Icon
            size={14}
            className={cn(isAlarmActive ? 'text-tertiary' : isFluidWarn ? 'text-amber-400' : config.color)}
          />
        </div>
        <span className="text-xs text-on-surface-dim uppercase tracking-widest font-medium">
          {probe.display_name}
        </span>
      </div>

      {isBinary ? (
        <div className="flex items-baseline gap-1.5 mb-1">
          <span
            className={cn(
              'text-2xl font-bold tracking-tight',
              isAlarmActive ? 'text-tertiary text-glow-tertiary' : isFluidWarn ? 'text-amber-400' : 'text-secondary',
            )}
          >
            {getBinaryLabel(probe)}
          </span>
        </div>
      ) : (
        <>
          <div className="flex items-baseline gap-1.5 mb-1">
            <span className={`text-4xl font-bold tracking-tight ${config.color} ${config.glowClass}`}>
              {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
            </span>
            <span className="text-lg text-on-surface-dim font-light">
              {probe.unit}
            </span>
          </div>
          <div className="h-[28px] mb-3" />
          <div className="h-10">
            <Sparkline data={sparklineData} color={config.sparklineColor} />
          </div>
        </>
      )}
    </button>
  )
}

export { getCategory, type ProbeCategory }

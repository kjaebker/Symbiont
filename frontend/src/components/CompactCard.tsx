import { useNavigate } from 'react-router-dom'
import { Thermometer, FlaskConical, Zap, ToggleLeft, Power, Utensils } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useMeasurements } from '@/hooks/useMeasurements'
import { useFeedStatus } from '@/hooks/useFeed'
import type { Probe, Outlet, Device } from '@/api/types'
import { getCategory } from './ProbeCard'

// --- Probe ---

const statusDot = {
  normal: 'bg-secondary',
  warning: 'bg-amber-400',
  critical: 'bg-tertiary',
  unknown: 'bg-on-surface-faint',
} as const

const categoryCompact = {
  temperature: {
    icon: Thermometer,
    color: 'text-tertiary',
    bg: 'bg-tertiary/10',
    hoverGlow: 'hover:shadow-glow-tertiary',
    glowClass: 'text-glow-tertiary',
  },
  chemistry: {
    icon: FlaskConical,
    color: 'text-secondary',
    bg: 'bg-secondary/10',
    hoverGlow: 'hover:shadow-glow-secondary',
    glowClass: 'text-glow-secondary',
  },
  power: {
    icon: Zap,
    color: 'text-primary',
    bg: 'bg-primary/10',
    hoverGlow: 'hover:shadow-glow-primary',
    glowClass: 'text-glow-primary',
  },
  digital: {
    icon: ToggleLeft,
    color: 'text-on-surface-dim',
    bg: 'bg-surface-container-high',
    hoverGlow: 'hover:shadow-glow-primary',
    glowClass: '',
  },
}

interface ProbeCompactCardProps {
  probe: Probe
}

export function ProbeCompactCard({ probe }: ProbeCompactCardProps) {
  const navigate = useNavigate()
  const category = getCategory(probe.type)
  const config = categoryCompact[category]
  const Icon = config.icon

  return (
    <button
      onClick={() => navigate(`/history?probe=${encodeURIComponent(probe.name)}`)}
      className={cn(
        'aspect-square bg-surface-container rounded-2xl p-3 flex flex-col items-center justify-center gap-1.5 transition-fluid hover:bg-surface-container-high cursor-pointer w-full',
        config.hoverGlow,
      )}
    >
      <div className={cn('w-9 h-9 rounded-xl flex items-center justify-center', config.bg)}>
        <Icon size={18} className={config.color} />
      </div>
      <span className={cn('text-lg font-bold text-on-surface leading-none', config.glowClass)}>
        {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
        <span className="text-xs font-normal text-on-surface-dim ml-0.5">{probe.unit}</span>
      </span>
      <span className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate w-full text-center">
        {probe.display_name}
      </span>
      <span
        className={cn(
          'h-1.5 w-1.5 rounded-full',
          statusDot[probe.status],
          probe.status === 'normal' && 'animate-bio-pulse',
        )}
      />
    </button>
  )
}

// --- Outlet ---

const stateColors: Record<string, string> = {
  ON: 'text-secondary',
  AON: 'text-primary',
  TBL: 'text-primary',
  OFF: 'text-on-surface-faint',
  AOF: 'text-on-surface-faint',
}

const stateLabels: Record<string, string> = {
  ON: 'On',
  OFF: 'Off',
  AON: 'Auto',
  AOF: 'Auto Off',
  TBL: 'Sched',
  PF1: 'Fail 1',
  PF2: 'Fail 2',
  PF3: 'Fail 3',
  PF4: 'Fail 4',
}

interface OutletCompactCardProps {
  outlet: Outlet
}

export function OutletCompactCard({ outlet }: OutletCompactCardProps) {
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'

  return (
    <div
      className={cn(
        'aspect-square bg-surface-container rounded-2xl p-3 flex flex-col items-center justify-center gap-1.5 transition-fluid w-full',
        isOn ? 'hover:shadow-glow-primary' : '',
      )}
    >
      <div
        className={cn(
          'w-9 h-9 rounded-xl flex items-center justify-center',
          isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
        )}
      >
        {isOn ? (
          <Zap size={18} className="text-primary" />
        ) : (
          <Power size={18} className="text-on-surface-faint" />
        )}
      </div>
      <span
        className={cn(
          'text-sm font-bold leading-none',
          stateColors[outlet.state] ?? 'text-on-surface-dim',
        )}
      >
        {stateLabels[outlet.state] ?? outlet.state}
      </span>
      <span className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate w-full text-center">
        {outlet.display_name || outlet.name}
      </span>
    </div>
  )
}

// --- Measurement ---

interface MeasurementCompactCardProps {
  parameter: string
}

export function MeasurementCompactCard({ parameter }: MeasurementCompactCardProps) {
  const navigate = useNavigate()
  const { data, isLoading } = useMeasurements({ parameter, limit: 1 })
  const latest = data?.measurements[0] ?? null

  const displayValue = latest
    ? latest.value % 1 === 0
      ? latest.value.toFixed(0)
      : latest.value.toFixed(2)
    : '—'

  return (
    <button
      onClick={() => navigate('/measurements')}
      className="aspect-square bg-surface-container rounded-2xl p-3 flex flex-col items-center justify-center gap-1.5 transition-fluid hover:bg-surface-container-high hover:shadow-glow-secondary cursor-pointer w-full"
    >
      <div className="w-9 h-9 rounded-xl flex items-center justify-center bg-secondary/10">
        <FlaskConical size={18} className="text-secondary" />
      </div>
      {isLoading ? (
        <div className="h-5 w-10 bg-surface-container-high rounded animate-pulse" />
      ) : (
        <span className="text-lg font-bold text-on-surface text-glow-secondary leading-none">
          {displayValue}
          {latest?.canonical_unit && (
            <span className="text-xs font-normal text-on-surface-dim ml-0.5">
              {latest.canonical_unit}
            </span>
          )}
        </span>
      )}
      <span className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate w-full text-center">
        {parameter}
      </span>
    </button>
  )
}

// --- Feed Mode ---

export function FeedCompactCard() {
  const { data } = useFeedStatus()
  const isActive = (data?.active ?? 0) === 1
  const activeFeed = data?.name ?? 0
  const feedLabel =
    isActive && activeFeed >= 1 && activeFeed <= 4
      ? ['A', 'B', 'C', 'D'][activeFeed - 1]
      : isActive
        ? 'On'
        : 'Off'

  return (
    <div
      className={cn(
        'aspect-square bg-surface-container rounded-2xl p-3 flex flex-col items-center justify-center gap-1.5 transition-fluid w-full',
        isActive && 'hover:shadow-glow-primary',
      )}
    >
      <div
        className={cn(
          'w-9 h-9 rounded-xl flex items-center justify-center',
          isActive ? 'bg-primary/10' : 'bg-surface-container-highest',
        )}
      >
        <Utensils size={18} className={isActive ? 'text-primary' : 'text-on-surface-faint'} />
      </div>
      <span
        className={cn(
          'text-sm font-bold leading-none',
          isActive ? 'text-primary' : 'text-on-surface-faint',
        )}
      >
        {feedLabel}
      </span>
      <span className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate w-full text-center">
        Feed
      </span>
      {isActive && (
        <span className="h-1.5 w-1.5 rounded-full bg-primary animate-bio-pulse" />
      )}
    </div>
  )
}

// --- Device (shows primary probe value) ---

interface DeviceCompactCardProps {
  device: Device
  primaryProbe?: Probe
}

export function DeviceCompactCard({ device, primaryProbe }: DeviceCompactCardProps) {
  const navigate = useNavigate()

  const Icon = primaryProbe ? categoryCompact[getCategory(primaryProbe.type)].icon : Zap
  const config = primaryProbe ? categoryCompact[getCategory(primaryProbe.type)] : categoryCompact.power

  return (
    <button
      onClick={() => primaryProbe
        ? navigate(`/history?probe=${encodeURIComponent(primaryProbe.name)}`)
        : undefined
      }
      className={cn(
        'aspect-square bg-surface-container rounded-2xl p-3 flex flex-col items-center justify-center gap-1.5 transition-fluid hover:bg-surface-container-high cursor-pointer w-full',
        config.hoverGlow,
      )}
    >
      <div className={cn('w-9 h-9 rounded-xl flex items-center justify-center', config.bg)}>
        <Icon size={18} className={config.color} />
      </div>
      {primaryProbe ? (
        <span className={cn('text-lg font-bold text-on-surface leading-none', config.glowClass)}>
          {primaryProbe.value.toFixed(primaryProbe.type === 'pH' ? 2 : 1)}
          <span className="text-xs font-normal text-on-surface-dim ml-0.5">{primaryProbe.unit}</span>
        </span>
      ) : (
        <span className="text-sm font-bold text-on-surface-faint leading-none">—</span>
      )}
      <span className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate w-full text-center">
        {device.name}
      </span>
      {primaryProbe && (
        <span
          className={cn(
            'h-1.5 w-1.5 rounded-full',
            statusDot[primaryProbe.status],
            primaryProbe.status === 'normal' && 'animate-bio-pulse',
          )}
        />
      )}
    </button>
  )
}

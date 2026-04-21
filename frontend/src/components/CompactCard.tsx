import { useNavigate } from 'react-router-dom'
import { Thermometer, FlaskConical, Zap, ToggleLeft, Power, Utensils } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useMeasurements } from '@/hooks/useMeasurements'
import { useFeedStatus } from '@/hooks/useFeed'
import type { Probe, Outlet, Device } from '@/api/types'
import { getCategory } from './ProbeCard'

// --- Category config ---

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
    glowClass: 'text-glow-tertiary',
    tint: 'rgba(255, 135, 150, 0.06)',
  },
  chemistry: {
    icon: FlaskConical,
    color: 'text-secondary',
    bg: 'bg-secondary/10',
    glowClass: 'text-glow-secondary',
    tint: 'rgba(109, 254, 156, 0.06)',
  },
  power: {
    icon: Zap,
    color: 'text-primary',
    bg: 'bg-primary/10',
    glowClass: 'text-glow-primary',
    tint: 'rgba(58, 223, 250, 0.06)',
  },
  digital: {
    icon: ToggleLeft,
    color: 'text-on-surface-dim',
    bg: 'bg-surface-container-high',
    glowClass: '',
    tint: 'transparent',
  },
}

// --- Probe ---

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
      onClick={() => navigate(`/history?tab=telemetry&probe=${encodeURIComponent(probe.name)}`)}
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 transition-fluid cursor-pointer w-full text-left"
      style={{
        background: `linear-gradient(135deg, var(--color-surface-container) 55%, ${config.tint})`,
      }}
    >
      <div className={cn('w-10 h-10 rounded-xl flex items-center justify-center shrink-0', config.bg)}>
        <Icon size={16} className={config.color} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-1 mb-0.5">
          <span className={`text-base font-bold leading-none ${config.color} ${config.glowClass}`}>
            {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
          </span>
          <span className="text-xs text-on-surface-dim font-normal leading-none">{probe.unit}</span>
        </div>
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate">
          {probe.display_name}
        </p>
      </div>
      <span
        className={cn(
          'h-1.5 w-1.5 rounded-full shrink-0',
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
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 w-full transition-fluid"
      style={{
        background: isOn
          ? 'linear-gradient(135deg, var(--color-surface-container) 55%, rgba(58,223,250,0.06))'
          : 'var(--color-surface-container)',
      }}
    >
      <div
        className={cn(
          'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
          isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
        )}
      >
        {isOn ? (
          <Zap size={16} className="text-primary" />
        ) : (
          <Power size={16} className="text-on-surface-faint" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p
          className={cn(
            'text-base font-bold leading-none mb-0.5',
            stateColors[outlet.state] ?? 'text-on-surface-dim',
          )}
        >
          {stateLabels[outlet.state] ?? outlet.state}
        </p>
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate">
          {outlet.display_name || outlet.name}
        </p>
      </div>
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
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 transition-fluid cursor-pointer w-full text-left"
      style={{
        background: 'linear-gradient(135deg, var(--color-surface-container) 55%, rgba(109,254,156,0.06))',
      }}
    >
      <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0 bg-secondary/10">
        <FlaskConical size={16} className="text-secondary" />
      </div>
      <div className="min-w-0 flex-1">
        {isLoading ? (
          <div className="h-4 w-12 bg-surface-container-high rounded animate-pulse mb-0.5" />
        ) : (
          <div className="flex items-baseline gap-1 mb-0.5">
            <span className="text-base font-bold leading-none text-secondary text-glow-secondary">
              {displayValue}
            </span>
            {latest?.canonical_unit && (
              <span className="text-xs text-on-surface-dim font-normal leading-none">
                {latest.canonical_unit}
              </span>
            )}
          </div>
        )}
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate">
          {parameter}
        </p>
      </div>
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
      ? `Feed ${['A', 'B', 'C', 'D'][activeFeed - 1]}`
      : isActive
        ? 'Active'
        : 'Inactive'

  return (
    <div
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 w-full transition-fluid"
      style={{
        background: isActive
          ? 'linear-gradient(135deg, var(--color-surface-container) 55%, rgba(58,223,250,0.06))'
          : 'var(--color-surface-container)',
      }}
    >
      <div
        className={cn(
          'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
          isActive ? 'bg-primary/10' : 'bg-surface-container-highest',
        )}
      >
        <Utensils size={16} className={isActive ? 'text-primary' : 'text-on-surface-faint'} />
      </div>
      <div className="min-w-0 flex-1">
        <p
          className={cn(
            'text-base font-bold leading-none mb-0.5',
            isActive ? 'text-primary' : 'text-on-surface-faint',
          )}
        >
          {feedLabel}
        </p>
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium">
          Feed Mode
        </p>
      </div>
      {isActive && (
        <span className="h-1.5 w-1.5 rounded-full bg-primary animate-bio-pulse shrink-0" />
      )}
    </div>
  )
}

// --- Device ---

interface DeviceCompactCardProps {
  device: Device
  primaryProbe?: Probe
}

export function DeviceCompactCard({ device, primaryProbe }: DeviceCompactCardProps) {
  const navigate = useNavigate()

  const config = primaryProbe
    ? categoryCompact[getCategory(primaryProbe.type)]
    : categoryCompact.power
  const Icon = config.icon

  return (
    <button
      onClick={() =>
        primaryProbe
          ? navigate(`/history?tab=telemetry&probe=${encodeURIComponent(primaryProbe.name)}`)
          : undefined
      }
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 transition-fluid cursor-pointer w-full text-left"
      style={{
        background: `linear-gradient(135deg, var(--color-surface-container) 55%, ${config.tint})`,
      }}
    >
      <div className={cn('w-10 h-10 rounded-xl flex items-center justify-center shrink-0', config.bg)}>
        <Icon size={16} className={config.color} />
      </div>
      <div className="min-w-0 flex-1">
        {primaryProbe ? (
          <div className="flex items-baseline gap-1 mb-0.5">
            <span className={`text-base font-bold leading-none ${config.color} ${config.glowClass}`}>
              {primaryProbe.value.toFixed(primaryProbe.type === 'pH' ? 2 : 1)}
            </span>
            <span className="text-xs text-on-surface-dim font-normal leading-none">{primaryProbe.unit}</span>
          </div>
        ) : (
          <p className="text-base font-bold text-on-surface-faint leading-none mb-0.5">—</p>
        )}
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate">
          {device.name}
        </p>
      </div>
      {primaryProbe && (
        <span
          className={cn(
            'h-1.5 w-1.5 rounded-full shrink-0',
            statusDot[primaryProbe.status],
            primaryProbe.status === 'normal' && 'animate-bio-pulse',
          )}
        />
      )}
    </button>
  )
}

import { useNavigate } from 'react-router-dom'
import { Thermometer, FlaskConical, Zap, ToggleLeft, Power, Utensils, Droplets, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useMeasurements } from '@/hooks/useMeasurements'
import { useFeedStatus } from '@/hooks/useFeed'
import type { Probe, Outlet, Device } from '@/api/types'
import { getCategory } from './ProbeCard'
import { inferPersonality } from '@/lib/devicePersonality'

// --- Category config ---

const categoryCompact = {
  temperature: {
    icon: Thermometer,
    color: 'text-tertiary',
    glowClass: 'text-glow-tertiary',
    hexColor: '#ff8796',
    tint: 'rgba(255, 135, 150, 0.18)',
    blobShape: '60% 40% 30% 70% / 60% 30% 70% 40%',
    blobBg: 'rgba(255, 135, 150, 0.12)',
  },
  chemistry: {
    icon: FlaskConical,
    color: 'text-secondary',
    glowClass: 'text-glow-secondary',
    hexColor: '#6dfe9c',
    tint: 'rgba(109, 254, 156, 0.18)',
    blobShape: '40% 60% 70% 30% / 50% 60% 40% 50%',
    blobBg: 'rgba(109, 254, 156, 0.10)',
  },
  power: {
    icon: Zap,
    color: 'text-primary',
    glowClass: 'text-glow-primary',
    hexColor: '#3adffa',
    tint: 'rgba(58, 223, 250, 0.18)',
    blobShape: '70% 30% 40% 60% / 40% 70% 30% 60%',
    blobBg: 'rgba(58, 223, 250, 0.10)',
  },
  digital: {
    icon: ToggleLeft,
    color: 'text-on-surface-dim',
    glowClass: '',
    hexColor: '#8a90a8',
    tint: 'rgba(138, 144, 168, 0.12)',
    blobShape: '50% 50% 60% 40% / 40% 60% 50% 50%',
    blobBg: 'rgba(138, 144, 168, 0.08)',
  },
}

// --- Probe ---

interface ProbeCompactCardProps {
  probe: Probe
}

function getBinaryCompactIcon(probe: Probe) {
  switch (probe.input_category) {
    case 'fluid': return Droplets
    case 'alarm': return AlertTriangle
    default: return ToggleLeft
  }
}

export function ProbeCompactCard({ probe }: ProbeCompactCardProps) {
  const navigate = useNavigate()
  const category = getCategory(probe.type)
  const config = categoryCompact[category]

  const isBinary = probe.is_binary
  const Icon = isBinary ? getBinaryCompactIcon(probe) : config.icon
  const isAlarmActive = probe.input_category === 'alarm' && probe.status === 'critical'
  const isFluidWarn = probe.input_category === 'fluid' && probe.status === 'warning'

  const iconColor = isAlarmActive ? 'text-tertiary' : isFluidWarn ? 'text-amber-400' : config.color

  const binaryLabel = probe.value !== 0 ? (probe.on_label ?? 'On') : (probe.off_label ?? 'Off')

  return (
    <button
      onClick={() => !isBinary && navigate(`/history?tab=telemetry&probe=${encodeURIComponent(probe.name)}`)}
      className={cn(
        'rounded-2xl px-3.5 py-3 flex items-center gap-3 transition-fluid w-full text-left relative overflow-hidden',
        isBinary ? 'cursor-default' : 'cursor-pointer',
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
          top: -16,
          right: -16,
          width: 80,
          height: 80,
          borderRadius: config.blobShape,
          background: isAlarmActive ? 'rgba(255,135,150,0.18)' : config.blobBg,
          filter: 'blur(16px)',
          pointerEvents: 'none',
        }}
      />
      <div
        className="w-10 h-10 flex items-center justify-center shrink-0"
        style={{
          borderRadius: isAlarmActive ? '60% 40% 30% 70% / 60% 30% 70% 40%' : config.blobShape,
          background: isAlarmActive ? 'rgba(255,135,150,0.22)'
            : (isFluidWarn || probe.status === 'warning') ? 'rgba(251,191,36,0.18)'
            : probe.status === 'critical' ? 'rgba(255,135,150,0.22)'
            : config.blobBg,
          boxShadow: isAlarmActive || probe.status === 'critical' ? '0 0 14px rgba(255,135,150,0.50)'
            : isFluidWarn || probe.status === 'warning' ? '0 0 12px rgba(251,191,36,0.40)'
            : `0 0 8px ${config.hexColor}1a`,
          animation: isAlarmActive || probe.status === 'critical' ? 'bioluminescent-pulse 1.8s ease-in-out infinite'
            : isFluidWarn || probe.status === 'warning' ? 'bioluminescent-pulse 2.8s ease-in-out infinite'
            : undefined,
        }}
      >
        <Icon size={16} className={iconColor} />
      </div>
      <div className="min-w-0 flex-1">
        {isBinary ? (
          <p className={cn('text-base font-bold leading-none mb-0.5', iconColor)}>
            {binaryLabel}
          </p>
        ) : (
          <div className="flex items-baseline gap-1 mb-0.5">
            <span className={`text-base font-bold leading-none ${config.color} ${config.glowClass}`}>
              {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
            </span>
            <span className="text-xs text-on-surface-dim font-normal leading-none">{probe.unit}</span>
          </div>
        )}
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate">
          {probe.display_name}
        </p>
      </div>
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
  const personality = inferPersonality(outlet.display_name || outlet.name, null)

  return (
    <div
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 w-full transition-fluid relative overflow-hidden"
      style={{
        background: `linear-gradient(135deg, var(--color-surface-container) 55%, ${personality.color}${isOn ? '22' : '0f'})`,
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: -16,
          right: -16,
          width: 80,
          height: 80,
          borderRadius: personality.blob,
          background: personality.bg,
          filter: 'blur(16px)',
          pointerEvents: 'none',
        }}
      />
      <div
        className="w-10 h-10 flex items-center justify-center shrink-0 transition-fluid"
        style={{
          borderRadius: personality.blob,
          background: isOn ? personality.bg : `${personality.color}0a`,
          boxShadow: isOn ? `0 0 10px ${personality.color}26` : `0 0 8px ${personality.color}1a`,
        }}
      >
        {isOn ? (
          <Zap size={16} style={{ color: personality.color }} />
        ) : (
          <Power size={16} style={{ color: personality.color, opacity: 0.5 }} />
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
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 transition-fluid cursor-pointer w-full text-left relative overflow-hidden"
      style={{
        background: 'linear-gradient(135deg, var(--color-surface-container) 55%, rgba(109,254,156,0.18))',
      }}
    >
      <div
        style={{
          position: 'absolute',
          top: -16,
          right: -16,
          width: 80,
          height: 80,
          borderRadius: '40% 60% 70% 30% / 50% 60% 40% 50%',
          background: 'rgba(109,254,156,0.10)',
          filter: 'blur(16px)',
          pointerEvents: 'none',
        }}
      />
      <div
        className="w-10 h-10 flex items-center justify-center shrink-0"
        style={{
          borderRadius: '40% 60% 70% 30% / 50% 60% 40% 50%',
          background: 'rgba(109,254,156,0.10)',
          boxShadow: '0 0 8px #6dfe9c1a',
        }}
      >
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
  const personality = inferPersonality(device.name, device.device_type)

  return (
    <button
      onClick={() =>
        primaryProbe
          ? navigate(`/history?tab=telemetry&probe=${encodeURIComponent(primaryProbe.name)}`)
          : undefined
      }
      className="rounded-2xl px-3.5 py-3 flex items-center gap-3 transition-fluid cursor-pointer w-full text-left"
      style={{
        background: `linear-gradient(135deg, var(--color-surface-container) 55%, ${personality.color}18)`,
      }}
    >
      <div
        className="w-10 h-10 flex items-center justify-center shrink-0 transition-fluid"
        style={{
          borderRadius: personality.blob,
          background: primaryProbe?.status === 'critical' ? 'rgba(255,135,150,0.22)'
            : primaryProbe?.status === 'warning' ? 'rgba(251,191,36,0.18)'
            : `${personality.color}0a`,
          boxShadow: primaryProbe?.status === 'critical' ? '0 0 14px rgba(255,135,150,0.50)'
            : primaryProbe?.status === 'warning' ? '0 0 12px rgba(251,191,36,0.40)'
            : 'none',
          animation: primaryProbe?.status === 'critical' ? 'bioluminescent-pulse 1.8s ease-in-out infinite'
            : primaryProbe?.status === 'warning' ? 'bioluminescent-pulse 2.8s ease-in-out infinite'
            : undefined,
        }}
      >
        <Zap size={16} style={{
          color: primaryProbe?.status === 'critical' ? '#ff8796'
            : primaryProbe?.status === 'warning' ? '#fbbf24'
            : personality.color,
        }} />
      </div>
      <div className="min-w-0 flex-1">
        {primaryProbe ? (
          <div className="flex items-baseline gap-1 mb-0.5">
            <span
              className="text-base font-bold leading-none"
              style={{ color: personality.color, textShadow: `0 0 6px ${personality.color}4d` }}
            >
              {primaryProbe.value.toFixed(primaryProbe.type === 'pH' ? 2 : 1)}
            </span>
            <span className="text-xs text-on-surface-dim font-normal leading-none">{primaryProbe.unit}</span>
          </div>
        ) : (
          <p className="text-base font-bold leading-none mb-0.5" style={{ color: personality.color, opacity: 0.5 }}>—</p>
        )}
        <p className="text-[10px] text-on-surface-faint uppercase tracking-widest font-medium truncate">
          {device.name}
        </p>
      </div>
    </button>
  )
}

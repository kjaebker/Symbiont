import { Flame, Waves, Sun, Filter, FlaskConical, Droplet, Snowflake, Fan, Zap, Power } from 'lucide-react'
import type { Outlet } from '@/api/types'
import { useSetOutlet } from '@/hooks/useOutlets'
import { cn } from '@/lib/utils'
import { inferPersonality } from '@/lib/devicePersonality'
import { cardGradient, CardIconBlob, OutletControlButtons } from './CardBase'

const stateLabels: Record<string, string> = {
  ON: 'ON',
  OFF: 'OFF',
  AON: 'AUTO',
  AOF: 'AUTO OFF',
  TBL: 'SCHEDULE',
  PF1: 'FAIL 1',
  PF2: 'FAIL 2',
  PF3: 'FAIL 3',
  PF4: 'FAIL 4',
}

const stateColors: Record<string, string> = {
  ON: 'text-secondary',
  AON: 'text-primary',
  OFF: 'text-on-surface-faint',
  AOF: 'text-on-surface-faint',
  TBL: 'text-primary',
}

const deviceIcons: Record<string, typeof Flame> = {
  heater: Flame,
  pump: Waves,
  wavemaker: Waves,
  light: Sun,
  skimmer: Filter,
  reactor: FlaskConical,
  chemistry: FlaskConical,
  doser: FlaskConical,
  ato: Droplet,
  chiller: Snowflake,
  fan: Fan,
}

interface OutletCardProps {
  outlet: Outlet
  controlsLocked?: boolean
}

export function OutletCard({ outlet, controlsLocked = false }: OutletCardProps) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  // outlet.type is Neptune port type ("outlet", "serial", etc.), not device category — use name
  const personality = inferPersonality(outlet.display_name || outlet.name, null)
  const DeviceIcon = deviceIcons[outlet.type] ?? (isOn ? Zap : Power)

  function handleControl(state: 'ON' | 'OFF' | 'AUTO') {
    if (controlsLocked) return
    mutation.mutate({ id: outlet.id, state })
  }

  return (
    <div
      className="rounded-2xl p-5 transition-fluid relative overflow-hidden"
      style={{ background: cardGradient(personality.color, isOn) }}
    >

      {/* Header: icon + name + bio-dot */}
      <div className="flex items-center justify-between mb-1 relative">
        <div className="flex items-center gap-3 min-w-0">
          <CardIconBlob
            color={personality.color}
            blobShape={personality.blob}
            isOn={isOn}
            activeBg={personality.bg}
            normalShadow={`0 0 8px ${personality.color}1a`}
          >
            <DeviceIcon size={18} style={{ color: isOn ? personality.color : 'var(--color-on-surface-faint)' }} />
          </CardIconBlob>
          <div className="min-w-0">
            <div className="text-sm font-semibold text-on-surface truncate">
              {outlet.display_name || outlet.name}
            </div>
            <div
              className="text-[10px] uppercase tracking-[0.12em] font-semibold"
              style={{ color: personality.color, opacity: 0.75 }}
            >
              {personality.label}
            </div>
          </div>
        </div>
        <span
          className={cn(
            'text-xs font-semibold uppercase tracking-wider shrink-0 ml-2',
            stateColors[outlet.state] ?? 'text-on-surface-dim',
          )}
        >
          {stateLabels[outlet.state] ?? outlet.state}
        </span>
      </div>

      <OutletControlButtons
        outletState={outlet.state}
        onControl={handleControl}
        isPending={mutation.isPending}
        isError={mutation.isError}
        revealed={!controlsLocked}
        innerPadding="pt-4"
      />
    </div>
  )
}

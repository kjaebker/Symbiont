import { Flame, Waves, Sun, Filter, FlaskConical, Droplet, Snowflake, Fan, Zap, Power, Box } from 'lucide-react'
import type { Outlet } from '@/api/types'
import { useSetOutlet } from '@/hooks/useOutlets'
import { cn } from '@/lib/utils'
import { inferPersonality } from '@/lib/devicePersonality'
import { BioDot } from './BioDot'

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

const statusDotHex: Record<string, string> = {
  on: '#6dfe9c',
  auto: '#3adffa',
  off: '#5a6080',
}

interface OutletCardProps {
  outlet: Outlet
  controlsLocked?: boolean
}

export function OutletCard({ outlet, controlsLocked = false }: OutletCardProps) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  const isAuto = outlet.state === 'AON' || outlet.state === 'AOF' || outlet.state === 'TBL'
  // outlet.type is Neptune port type ("outlet", "serial", etc.), not device category — use name
  const personality = inferPersonality(outlet.display_name || outlet.name, null)
  const DeviceIcon = deviceIcons[outlet.type] ?? (isOn ? Zap : Power)

  const dotColor =
    outlet.state === 'ON' ? statusDotHex.on
    : isAuto ? statusDotHex.auto
    : statusDotHex.off

  function handleControl(state: 'ON' | 'OFF' | 'AUTO') {
    if (controlsLocked) return
    mutation.mutate({ id: outlet.id, state })
  }

  return (
    <div
      className="rounded-2xl p-5 transition-fluid relative overflow-hidden"
      style={{
        background: isOn
          ? `linear-gradient(135deg, var(--color-surface-container) 40%, ${personality.bg})`
          : 'var(--color-surface-container)',
      }}
    >
      {/* Personality blob */}
      <div
        style={{
          position: 'absolute',
          top: -20,
          right: -20,
          width: 120,
          height: 120,
          borderRadius: personality.blob,
          background: personality.bg,
          filter: 'blur(20px)',
          pointerEvents: 'none',
        }}
      />

      {/* Header: icon + name + bio-dot */}
      <div className="flex items-center justify-between mb-1 relative">
        <div className="flex items-center gap-3 min-w-0">
          <div
            className={cn(
              'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
              isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
            )}
            style={isOn ? { backgroundColor: `${personality.color}18` } : undefined}
          >
            <DeviceIcon size={18} style={{ color: isOn ? personality.color : 'var(--color-on-surface-faint)' }} />
          </div>
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
        <div className="flex items-center gap-2 shrink-0 ml-2">
          <span
            className={cn(
              'text-xs font-semibold uppercase tracking-wider',
              stateColors[outlet.state] ?? 'text-on-surface-dim',
            )}
          >
            {stateLabels[outlet.state] ?? outlet.state}
          </span>
          <BioDot color={dotColor} size={10} />
        </div>
      </div>

      {/* Controls — revealed when unlocked via grid-template-rows animation */}
      <div
        className="grid overflow-hidden relative"
        style={{
          gridTemplateRows: controlsLocked ? '0fr' : '1fr',
          transition: 'grid-template-rows 250ms cubic-bezier(0.65, 0, 0.35, 1)',
        }}
      >
        <div className="min-h-0">
          <div className="flex gap-1.5 pt-4">
            {(['OFF', 'ON', 'AUTO'] as const).map((s) => {
              const active =
                (s === 'OFF' && (outlet.state === 'OFF' || outlet.state === 'AOF')) ||
                (s === 'ON' && outlet.state === 'ON') ||
                (s === 'AUTO' && isAuto)

              return (
                <button
                  key={s}
                  onClick={() => handleControl(s)}
                  disabled={mutation.isPending}
                  className={cn(
                    'flex-1 py-2 rounded-xl text-xs font-semibold uppercase tracking-wider transition-fluid',
                    active
                      ? s === 'AUTO'
                        ? 'bg-primary text-on-primary'
                        : s === 'ON'
                          ? 'bg-secondary text-on-secondary'
                          : 'bg-surface-container-highest text-on-surface'
                      : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container-highest',
                  )}
                >
                  {s}
                </button>
              )
            })}
          </div>
          {mutation.isError && (
            <p className="text-xs text-tertiary mt-2">Failed to set state. Try again.</p>
          )}
        </div>
      </div>
    </div>
  )
}

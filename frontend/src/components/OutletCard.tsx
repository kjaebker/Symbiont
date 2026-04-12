import { Power, Zap } from 'lucide-react'
import type { Outlet } from '@/api/types'
import { useSetOutlet } from '@/hooks/useOutlets'
import { cn } from '@/lib/utils'

const stateLabels: Record<string, string> = {
  ON: 'On',
  OFF: 'Off',
  AON: 'Auto',
  AOF: 'Auto Off',
  TBL: 'Schedule',
  PF1: 'Fail 1',
  PF2: 'Fail 2',
  PF3: 'Fail 3',
  PF4: 'Fail 4',
}

const stateColors: Record<string, string> = {
  ON: 'text-secondary',
  AON: 'text-primary',
  OFF: 'text-on-surface-faint',
  AOF: 'text-on-surface-faint',
  TBL: 'text-primary',
}

interface OutletCardProps {
  outlet: Outlet
  controlsLocked?: boolean
}

export function OutletCard({ outlet, controlsLocked = false }: OutletCardProps) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  const isAuto = outlet.state === 'AON' || outlet.state === 'AOF' || outlet.state === 'TBL'

  function handleControl(state: 'ON' | 'OFF' | 'AUTO') {
    if (controlsLocked) return
    mutation.mutate({ id: outlet.id, state })
  }

  return (
    <div
      className="rounded-2xl p-5 transition-fluid"
      style={{
        background: isOn
          ? 'linear-gradient(135deg, var(--color-surface-container) 40%, rgba(58,223,250,0.07))'
          : 'var(--color-surface-container)',
      }}
    >
      {/* Header: icon + name + state label */}
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-3 min-w-0">
          <div
            className={cn(
              'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
              isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
            )}
          >
            {isOn ? (
              <Zap size={18} className="text-primary" />
            ) : (
              <Power size={18} className="text-on-surface-faint" />
            )}
          </div>
          <span className="text-sm font-semibold text-on-surface truncate">
            {outlet.display_name || outlet.name}
          </span>
        </div>
        <span
          className={cn(
            'text-xs font-semibold uppercase tracking-wider shrink-0 ml-2 mt-0.5',
            stateColors[outlet.state] ?? 'text-on-surface-dim',
          )}
        >
          {stateLabels[outlet.state] ?? outlet.state}
        </span>
      </div>

      {/* Controls — revealed when unlocked via grid-template-rows animation */}
      <div
        className="grid overflow-hidden"
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

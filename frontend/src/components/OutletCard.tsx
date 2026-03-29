import { Lock, Power, Zap } from 'lucide-react'
import type { Outlet } from '@/api/types'
import { useSetOutlet } from '@/hooks/useOutlets'
import { cn } from '@/lib/utils'

// Apex reports AON (Auto On) and AOF (Auto Off) for outlets under program control.
// We display both as "Auto" variants since the ON/OFF is the program's decision, not the user's.
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
  OFF: 'text-on-surface-dim',
  AOF: 'text-on-surface-dim',
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
    <div className="bg-surface-container rounded-2xl p-5 transition-fluid">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'w-10 h-10 rounded-xl flex items-center justify-center',
              isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
            )}
          >
            {isOn ? (
              <Zap size={18} className="text-primary" />
            ) : (
              <Power size={18} className="text-on-surface-faint" />
            )}
          </div>
          <div>
            <div className="text-sm font-semibold text-on-surface">
              {outlet.display_name || outlet.name}
            </div>
            <div
              className={cn(
                'text-xs font-medium uppercase tracking-wider',
                stateColors[outlet.state] ?? 'text-on-surface-dim',
              )}
            >
              {stateLabels[outlet.state] ?? outlet.state}
            </div>
          </div>
        </div>
      </div>

      <div className={cn('relative', controlsLocked && 'opacity-40')}>
        <div className="flex gap-1.5">
          {(['OFF', 'ON', 'AUTO'] as const).map((s) => {
            const active =
              (s === 'OFF' && (outlet.state === 'OFF' || outlet.state === 'AOF')) ||
              (s === 'ON' && outlet.state === 'ON') ||
              (s === 'AUTO' && isAuto)

            return (
              <button
                key={s}
                onClick={() => handleControl(s)}
                disabled={mutation.isPending || controlsLocked}
                className={cn(
                  'flex-1 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid',
                  controlsLocked
                    ? 'bg-surface-container-high text-on-surface-faint cursor-not-allowed'
                    : active
                      ? s === 'AUTO'
                        ? 'bg-primary text-on-primary'
                        : s === 'ON'
                          ? 'bg-secondary text-on-secondary'
                          : 'bg-surface-container-highest text-on-surface'
                      : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                )}
              >
                {s}
              </button>
            )
          })}
        </div>
        {controlsLocked && (
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <Lock size={14} className="text-on-surface-dim" />
          </div>
        )}
      </div>
      {mutation.isError && !controlsLocked && (
        <p className="text-xs text-tertiary mt-2">
          Failed to set state. Try again.
        </p>
      )}
    </div>
  )
}

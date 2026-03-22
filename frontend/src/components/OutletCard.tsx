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
  OFF: 'text-on-surface-dim',
  AOF: 'text-on-surface-dim',
  TBL: 'text-primary',
}

interface OutletCardProps {
  outlet: Outlet
}

export function OutletCard({ outlet }: OutletCardProps) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'

  function handleControl(state: 'ON' | 'OFF') {
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

      <div className="flex gap-1.5">
        {(['OFF', 'ON'] as const).map((s) => {
          const active =
            (s === 'OFF' && (outlet.state === 'OFF' || outlet.state === 'AOF')) ||
            (s === 'ON' && (outlet.state === 'ON' || outlet.state === 'AON'))

          return (
            <button
              key={s}
              onClick={() => handleControl(s)}
              disabled={mutation.isPending}
              className={cn(
                'flex-1 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid',
                active
                  ? s === 'ON'
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
    </div>
  )
}

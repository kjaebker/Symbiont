import { useState } from 'react'
import { Power, Zap } from 'lucide-react'
import { useOutlets, useSetOutlet, useOutletEvents } from '@/hooks/useOutlets'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import { relativeTime } from '@/lib/utils'
import type { Outlet } from '@/api/types'

const stateLabels: Record<string, string> = {
  ON: 'On',
  OFF: 'Off',
  AON: 'Auto On',
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
  PF1: 'text-tertiary',
  PF2: 'text-tertiary',
  PF3: 'text-tertiary',
  PF4: 'text-tertiary',
}

const initiatedByColors: Record<string, string> = {
  ui: 'bg-primary/15 text-primary',
  cli: 'bg-secondary/15 text-secondary',
  mcp: 'bg-tertiary/15 text-tertiary',
  api: 'bg-on-surface-faint/15 text-on-surface-dim',
}

function OutletRow({ outlet }: { outlet: Outlet }) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  const isAuto = outlet.state === 'AON' || outlet.state === 'AOF' || outlet.state === 'TBL'

  function handleControl(state: 'ON' | 'OFF' | 'AUTO') {
    mutation.mutate({ id: outlet.id, state })
  }

  return (
    <tr className="group transition-fluid hover:bg-surface-container-high/50">
      <td className="py-3 px-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0',
              isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
            )}
          >
            {isOn ? (
              <Zap size={14} className="text-primary" />
            ) : (
              <Power size={14} className="text-on-surface-faint" />
            )}
          </div>
          <span className="text-sm font-medium text-on-surface">
            {outlet.display_name || outlet.name}
          </span>
        </div>
      </td>
      <td className="py-3 px-4">
        <span
          className={cn(
            'text-xs font-semibold uppercase tracking-wider',
            stateColors[outlet.state] ?? 'text-on-surface-dim',
          )}
        >
          {stateLabels[outlet.state] ?? outlet.state}
        </span>
      </td>
      <td className="py-3 px-4">
        <span className="text-xs text-on-surface-dim uppercase tracking-wider">
          {outlet.type}
        </span>
      </td>
      <td className="py-3 px-4">
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
                disabled={mutation.isPending}
                className={cn(
                  'px-3 py-1 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
                  active
                    ? s === 'AUTO'
                      ? 'bg-primary text-on-primary'
                      : s === 'ON'
                        ? 'bg-secondary text-on-secondary'
                        : 'bg-surface-container-highest text-on-surface'
                    : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                  mutation.isPending && 'opacity-50 cursor-not-allowed',
                )}
              >
                {s}
              </button>
            )
          })}
        </div>
        {mutation.isError && (
          <p className="text-xs text-tertiary mt-1">
            Failed to set state
          </p>
        )}
      </td>
    </tr>
  )
}

export default function Outlets() {
  usePageTitle('Outlets')
  const { data, isLoading } = useOutlets()
  const [eventLimit, setEventLimit] = useState(50)
  const [sourceFilter, setSourceFilter] = useState<string>('')
  const { data: eventsData } = useOutletEvents({
    limit: eventLimit,
    ...(sourceFilter && { initiated_by: sourceFilter }),
  })

  const outlets = data?.outlets ?? []
  const events = eventsData?.events ?? []

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <p className="text-xs text-primary uppercase tracking-widest mb-2">
          Energy Command
        </p>
        <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
          Outlet Control
        </h1>
      </div>

      {/* Outlet Table */}
      <div className="bg-surface-container rounded-2xl overflow-hidden">
        {isLoading ? (
          <div className="p-12 flex items-center justify-center">
            <div className="flex flex-col items-center gap-3">
              <div className="h-8 w-8 rounded-full bg-primary/20 animate-bio-pulse" />
              <span className="text-xs text-on-surface-faint uppercase tracking-widest">
                Loading outlets...
              </span>
            </div>
          </div>
        ) : outlets.length === 0 ? (
          <div className="p-12 text-center">
            <span className="text-on-surface-faint text-sm">
              No outlets found
            </span>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-container-high/50">
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Name
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    State
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Type
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Control
                  </th>
                </tr>
              </thead>
              <tbody>
                {outlets.map((outlet) => (
                  <OutletRow key={outlet.id} outlet={outlet} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Event Log */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold text-on-surface">
            Recent Events
          </h2>
          <select
            value={sourceFilter}
            onChange={(e) => {
              setSourceFilter(e.target.value)
              setEventLimit(50)
            }}
            className="bg-surface-container-high text-on-surface text-xs rounded-lg px-3 py-1.5 uppercase tracking-wider focus:outline-none focus:ring-1 focus:ring-primary/50"
          >
            <option value="">All sources</option>
            <option value="ui">UI</option>
            <option value="cli">CLI</option>
            <option value="mcp">MCP</option>
            <option value="api">API</option>
          </select>
        </div>
        <div className="bg-surface-container rounded-2xl overflow-hidden">
          {events.length === 0 ? (
            <div className="p-8 text-center">
              <span className="text-on-surface-faint text-sm">
                No outlet events recorded yet
              </span>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="bg-surface-container-high/50">
                      <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                        Time
                      </th>
                      <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                        Outlet
                      </th>
                      <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                        Change
                      </th>
                      <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                        Source
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {events.map((event) => (
                      <tr
                        key={event.id}
                        className="transition-fluid hover:bg-surface-container-high/50"
                      >
                        <td className="py-2.5 px-4 text-xs text-on-surface-dim whitespace-nowrap">
                          {relativeTime(event.ts)}
                        </td>
                        <td className="py-2.5 px-4 text-sm text-on-surface">
                          {event.outlet_display_name ?? event.outlet_name}
                        </td>
                        <td className="py-2.5 px-4">
                          <span className="text-xs text-on-surface-dim">
                            <span className={stateColors[event.from_state] ?? 'text-on-surface-dim'}>
                              {stateLabels[event.from_state] ?? event.from_state}
                            </span>
                            <span className="text-on-surface-faint mx-1.5">&rarr;</span>
                            <span className={stateColors[event.to_state] ?? 'text-on-surface-dim'}>
                              {stateLabels[event.to_state] ?? event.to_state}
                            </span>
                          </span>
                        </td>
                        <td className="py-2.5 px-4">
                          <span
                            className={cn(
                              'inline-block px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider',
                              initiatedByColors[event.initiated_by] ?? 'bg-surface-container-high text-on-surface-dim',
                            )}
                          >
                            {event.initiated_by}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {events.length >= eventLimit && (
                <div className="p-3 flex justify-center">
                  <button
                    onClick={() => setEventLimit((l) => l + 50)}
                    className="text-xs text-primary hover:text-primary/80 uppercase tracking-wider font-medium transition-fluid"
                  >
                    Show more
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

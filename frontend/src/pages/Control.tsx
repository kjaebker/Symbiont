import { useState, useMemo } from 'react'
import { Power, Zap, ChevronDown } from 'lucide-react'
import { useOutlets, useSetOutlet } from '@/hooks/useOutlets'
import { useAuditEvents } from '@/hooks/useEvents'
import { usePageTitle } from '@/hooks/usePageTitle'
import { usePrograms, useSyncPrograms } from '@/hooks/usePrograms'
import { RefreshCw } from 'lucide-react'
import { cn, relativeTime } from '@/lib/utils'
import { ProgramView } from '@/components/ProgramView'
import { Button, Card, Chip, EmptyState, LoadingState, PageBody, SectionHeader, Select, labelClass } from '@/components/ui'
import type { ChipTone } from '@/components/ui'
import type { Outlet, AuditEvent, OutputProgram } from '@/api/client'

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

const stateTones: Record<string, ChipTone> = {
  ON: 'secondary',
  AON: 'primary',
  OFF: 'neutral',
  AOF: 'neutral',
  TBL: 'primary',
  PF1: 'critical',
  PF2: 'critical',
  PF3: 'critical',
  PF4: 'critical',
}

const initiatedByTones: Record<string, ChipTone> = {
  ui: 'primary',
  cli: 'secondary',
  // Coral is reserved for critical/destructive, so the assistant is neutral.
  mcp: 'neutral',
  api: 'neutral',
}

function ControlButtons({
  outlet,
  onControl,
  isPending,
}: {
  outlet: Outlet
  onControl: (s: 'ON' | 'OFF' | 'AUTO') => void
  isPending: boolean
}) {
  const isAuto = outlet.state === 'AON' || outlet.state === 'AOF' || outlet.state === 'TBL'
  return (
    <>
      {(['OFF', 'ON', 'AUTO'] as const).map((s) => {
        const active =
          (s === 'OFF' && (outlet.state === 'OFF' || outlet.state === 'AOF')) ||
          (s === 'ON' && outlet.state === 'ON') ||
          (s === 'AUTO' && isAuto)
        return (
          <button
            key={s}
            onClick={() => onControl(s)}
            disabled={isPending}
            className={cn(
              'px-3 py-1 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
              active
                ? s === 'AUTO'
                  ? 'bg-primary text-on-primary'
                  : s === 'ON'
                    ? 'bg-secondary text-on-secondary'
                    : 'bg-surface-container-highest text-on-surface'
                : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
              isPending && 'opacity-50 cursor-not-allowed',
            )}
          >
            {s}
          </button>
        )
      })}
    </>
  )
}

function OutletCard({ outlet, programsByDID }: { outlet: Outlet; programsByDID: Map<string, OutputProgram> }) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  const [showProgram, setShowProgram] = useState(false)
  const program = programsByDID.get(outlet.id)

  return (
    <div className="px-4 py-3">
      <div className="flex items-center justify-between gap-3 mb-3">
        <div className="flex items-center gap-3 min-w-0">
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
          <div className="min-w-0">
            <span className="text-sm font-medium text-on-surface block truncate">
              {outlet.display_name || outlet.name}
            </span>
            <span className="text-xs text-on-surface-dim uppercase tracking-wider">
              {outlet.type}
            </span>
          </div>
        </div>
        <Chip tone={stateTones[outlet.state] ?? 'neutral'} className="shrink-0">
          {stateLabels[outlet.state] ?? outlet.state}
        </Chip>
      </div>
      <div className="flex gap-1.5 items-center flex-wrap pl-11">
        <ControlButtons
          outlet={outlet}
          onControl={(s) => mutation.mutate({ id: outlet.id, state: s })}
          isPending={mutation.isPending}
        />
        {program && (
          <Button
            variant={showProgram ? 'tonal' : 'secondary'}
            size="sm"
            className="ml-1 uppercase tracking-wider gap-1"
            onClick={() => setShowProgram((v) => !v)}
          >
            Prog
            <ChevronDown
              size={11}
              className={cn('transition-transform duration-200', showProgram && 'rotate-180')}
            />
          </Button>
        )}
      </div>
      {mutation.isError && (
        <p className="text-xs text-tertiary mt-1 pl-11">Failed to set state</p>
      )}
      {program && showProgram && (
        <div className="mt-3 pt-3 pl-11">
          <ProgramView program={program} />
        </div>
      )}
    </div>
  )
}

function OutletRow({ outlet, programsByDID }: { outlet: Outlet; programsByDID: Map<string, OutputProgram> }) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  const [showProgram, setShowProgram] = useState(false)
  const program = programsByDID.get(outlet.id)

  return (
    <>
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
          <Chip tone={stateTones[outlet.state] ?? 'neutral'}>
            {stateLabels[outlet.state] ?? outlet.state}
          </Chip>
        </td>
        <td className="py-3 px-4">
          <span className="text-xs text-on-surface-dim uppercase tracking-wider">
            {outlet.type}
          </span>
        </td>
        <td className="py-3 px-4">
          <div className="flex gap-1.5 items-center">
            <ControlButtons
              outlet={outlet}
              onControl={(s) => mutation.mutate({ id: outlet.id, state: s })}
              isPending={mutation.isPending}
            />
            {program && (
              <Button
                variant={showProgram ? 'tonal' : 'secondary'}
                size="sm"
                className="ml-1 uppercase tracking-wider gap-1"
                onClick={() => setShowProgram((v) => !v)}
                title="View program"
              >
                Prog
                <ChevronDown
                  size={11}
                  className={cn('transition-transform duration-200', showProgram && 'rotate-180')}
                />
              </Button>
            )}
          </div>
          {mutation.isError && (
            <p className="text-xs text-tertiary mt-1">Failed to set state</p>
          )}
        </td>
      </tr>
      {program && showProgram && (
        <tr>
          <td colSpan={4} className="px-4 pb-4 pt-0 bg-surface-container-high/30">
            <div className="pl-11 pr-2 pt-3">
              <ProgramView program={program} />
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

function parseOutletPayload(event: AuditEvent) {
  try {
    const w = JSON.parse(event.payload_json) as { data: Record<string, string> }
    return w.data
  } catch {
    return null
  }
}

export default function Control({ hideHeader = false }: { hideHeader?: boolean }) {
  usePageTitle('Control')
  const { data, isLoading } = useOutlets()
  const { data: programsData } = usePrograms()
  const syncMutation = useSyncPrograms()
  const [eventLimit, setEventLimit] = useState(50)
  const [sourceFilter, setSourceFilter] = useState<string>('')
  const { data: eventsData } = useAuditEvents({
    kind: 'outlet_changed',
    limit: eventLimit,
    ...(sourceFilter && { initiated_by: sourceFilter }),
  })

  const outlets = data?.outlets ?? []
  const programsByDID = useMemo(
    () => new Map((programsData?.programs ?? []).map((p) => [p.did, p])),
    [programsData],
  )
  const outletDisplayNames = Object.fromEntries(
    outlets.map((o) => [o.id, o.display_name || o.name]),
  )
  const events = eventsData?.events ?? []

  return (
    <PageBody>
      {/* Header */}
      {!hideHeader && (
        <div>
          <p className="text-xs text-primary uppercase tracking-widest mb-2">
            Energy Command
          </p>
          <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
            Outlet Control
          </h1>
        </div>
      )}

      {/* Outlet Section */}
      <div className="space-y-3 sm:space-y-0 sm:bg-surface-container sm:rounded-2xl sm:overflow-hidden">
        {/* Program sync status bar */}
        <div className="flex items-center justify-between px-4 py-2.5 bg-surface-container rounded-2xl sm:rounded-none sm:bg-surface-container-high/30">
          <span className="text-xs text-on-surface-faint">
            {programsData?.synced_at
              ? <><span className="hidden sm:inline">Programs synced </span><span className="text-on-surface-dim">{relativeTime(programsData.synced_at)}</span></>
              : <span className="text-on-surface-faint/50">Programs not yet synced</span>}
          </span>
          <Button
            variant="ghost"
            size="sm"
            className="uppercase tracking-wider"
            onClick={() => syncMutation.mutate()}
            disabled={syncMutation.isPending}
          >
            <RefreshCw size={12} className={cn(syncMutation.isPending && 'animate-spin')} />
            {syncMutation.isPending ? 'Syncing…' : syncMutation.data ? `${syncMutation.data.changed} changed` : 'Sync Programs'}
          </Button>
        </div>

        {isLoading ? (
          <LoadingState label="Loading outlets..." />
        ) : outlets.length === 0 ? (
          <EmptyState message="No outlets found" />
        ) : (
          <>
            {/* Mobile: individual cards */}
            <div className="sm:hidden space-y-3">
              {outlets.map((outlet) => (
                <Card key={outlet.id} flush>
                  <OutletCard outlet={outlet} programsByDID={programsByDID} />
                </Card>
              ))}
            </div>

            {/* Desktop table */}
            <div className="hidden sm:block overflow-x-auto">
              <table className="w-full min-w-[580px]">
                <thead>
                  <tr className="bg-surface-container-high/50">
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      Name
                    </th>
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      State
                    </th>
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      Type
                    </th>
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      Control / Program
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {outlets.map((outlet) => (
                    <OutletRow key={outlet.id} outlet={outlet} programsByDID={programsByDID} />
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>

      {/* Event Log */}
      <div>
        <SectionHeader
          title="Recent Events"
          className="mb-3"
          action={
            <Select
              value={sourceFilter}
              onChange={(e) => {
                setSourceFilter(e.target.value)
                setEventLimit(50)
              }}
              aria-label="Filter by source"
              className="w-auto"
            >
              <option value="">All sources</option>
              <option value="ui">UI</option>
              <option value="cli">CLI</option>
              <option value="mcp">MCP</option>
              <option value="api">API</option>
            </Select>
          }
        />
        <Card flush>
          {events.length === 0 ? (
            <EmptyState message="No outlet events recorded yet" />
          ) : (
            <>
              <table className="w-full">
                <thead>
                  <tr className="bg-surface-container-high/50">
                    <th className={cn(labelClass, 'hidden sm:table-cell text-left py-3 px-4')}>
                      Time
                    </th>
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      Outlet
                    </th>
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      Change
                    </th>
                    <th className={cn(labelClass, 'text-left py-3 px-4')}>
                      Source
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {events.map((event) => {
                    const d = parseOutletPayload(event)
                    const displayName =
                      (event.correlation_id && outletDisplayNames[event.correlation_id]) ||
                      d?.name ||
                      event.correlation_id ||
                      '—'
                    const fromState = d?.prev_state ?? ''
                    const toState = d?.new_state ?? ''
                    const initiatedBy = d?.initiated_by ?? ''
                    return (
                      <tr
                        key={event.id}
                        className="transition-fluid hover:bg-surface-container-high/50"
                      >
                        <td className="hidden sm:table-cell py-2.5 px-4 text-xs text-on-surface-dim whitespace-nowrap">
                          {relativeTime(event.ts)}
                        </td>
                        <td className="py-2.5 px-4">
                          <span className="text-sm text-on-surface block">{displayName}</span>
                          <span className="sm:hidden text-xs text-on-surface-dim">
                            {relativeTime(event.ts)}
                          </span>
                        </td>
                        <td className="py-2.5 px-4">
                          <span className="text-xs text-on-surface-dim">
                            <span className={stateColors[fromState] ?? 'text-on-surface-dim'}>
                              {(stateLabels[fromState] ?? fromState) || '—'}
                            </span>
                            <span className="text-on-surface-faint mx-1.5">&rarr;</span>
                            <span className={stateColors[toState] ?? 'text-on-surface-dim'}>
                              {stateLabels[toState] ?? toState}
                            </span>
                          </span>
                        </td>
                        <td className="py-2.5 px-4">
                          <Chip tone={initiatedByTones[initiatedBy] ?? 'neutral'}>
                            {initiatedBy || '—'}
                          </Chip>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
              {events.length >= eventLimit && (
                <div className="p-3 flex justify-center">
                  <Button variant="ghost" size="sm" onClick={() => setEventLimit((l) => l + 50)}>
                    Show more
                  </Button>
                </div>
              )}
            </>
          )}
        </Card>
      </div>
    </PageBody>
  )
}

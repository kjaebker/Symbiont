import { useState, useMemo } from 'react'
import { Power, Zap, ChevronDown } from 'lucide-react'
import { RefreshCw } from 'lucide-react'
import { useOutlets, useSetOutlet } from '@/hooks/useOutlets'
import { useOutletConfigs, useUpdateOutletConfig } from '@/hooks/useSettings'
import { usePrograms, useSyncPrograms } from '@/hooks/usePrograms'
import { cn } from '@/lib/utils'
import { ProgramView } from '@/components/ProgramView'
import { Button, Card, Chip, EmptyState, LoadingState, labelClass, TimeAgo } from '@/components/ui'
import type { ChipTone } from '@/components/ui'
import type { Outlet, OutputProgram } from '@/api/client'
import { EditableCell } from './_shared'

// =============================================================================
// Outlets Tab: every outlet with its display name, state, manual override and
// program. The Dashboard only shows the outlets placed on its layout.
// =============================================================================

interface OutletItemProps {
  outlet: Outlet
  /** Configured display name, falling back to the Apex name. */
  displayName: string
  onRename: (name: string) => void
  programsByDID: Map<string, OutputProgram>
}

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

function OutletCard({ outlet, displayName, onRename, programsByDID }: OutletItemProps) {
  const mutation = useSetOutlet()
  const isOn = outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
  const [showProgram, setShowProgram] = useState(false)
  const program = programsByDID.get(outlet.id)

  return (
    <div className="px-4 py-3">
      <div className="flex items-center justify-between gap-3 mb-3">
        <div className="flex items-center gap-3 min-w-0 flex-1">
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
          <div className="min-w-0 flex-1">
            <EditableCell
              value={displayName}
              onSave={onRename}
              className="text-sm font-medium text-on-surface truncate"
            />
            <span className="px-2 text-xs text-on-surface-dim uppercase tracking-wider">
              {outlet.type} · {outlet.name}
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

function OutletRow({ outlet, displayName, onRename, programsByDID }: OutletItemProps) {
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
            <div className="min-w-0 flex-1">
              <EditableCell
                value={displayName}
                onSave={onRename}
                className="text-sm font-medium text-on-surface"
              />
              <span className="px-2 text-xs text-on-surface-faint font-mono">
                {outlet.name} · {outlet.id}
              </span>
            </div>
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

export default function OutletsTab() {
  const { data, isLoading } = useOutlets()
  const { data: configData } = useOutletConfigs()
  const { data: programsData } = usePrograms()
  const syncMutation = useSyncPrograms()
  const updateConfig = useUpdateOutletConfig()

  const outlets = data?.outlets ?? []
  const programsByDID = useMemo(
    () => new Map((programsData?.programs ?? []).map((p) => [p.did, p])),
    [programsData],
  )
  const configNames = new Map((configData?.configs ?? []).map((c) => [c.outlet_id, c.display_name]))

  const itemProps = (outlet: Outlet): OutletItemProps => ({
    outlet,
    displayName: configNames.get(outlet.id) || outlet.display_name || outlet.name,
    onRename: (name) => updateConfig.mutate({ id: outlet.id, config: { display_name: name } }),
    programsByDID,
  })

  return (
    <div className="space-y-3 sm:space-y-0">
      {/* Program sync status bar */}
      <div className="flex items-center justify-between px-4 py-2.5 bg-surface-container rounded-2xl sm:rounded-none sm:bg-surface-container-high/30">
        <span className="text-xs text-on-surface-faint">
          {programsData?.synced_at
            ? <><span className="hidden sm:inline">Programs synced </span><span className="text-on-surface-dim"><TimeAgo ts={programsData.synced_at} /></span></>
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
        <EmptyState message="No outlets found. Outlets will appear here after the first poll." />
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {outlets.map((outlet) => (
              <Card key={outlet.id} flush>
                <OutletCard {...itemProps(outlet)} />
              </Card>
            ))}
          </div>

          {/* Desktop table */}
          <div className="hidden sm:block overflow-x-auto">
            <table className="w-full min-w-[580px]">
              <thead>
                <tr className="bg-surface-container-high/50">
                  {['Name', 'State', 'Type', 'Control / Program'].map((h) => (
                    <th key={h} className={cn(labelClass, 'text-left py-3 px-4')}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {outlets.map((outlet) => (
                  <OutletRow key={outlet.id} {...itemProps(outlet)} />
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}

import { useProbes } from '@/hooks/useProbes'
import { useOutlets, useOutletEvents } from '@/hooks/useOutlets'
import { useSystemStatus } from '@/hooks/useSystem'
import { ProbeCard } from '@/components/ProbeCard'
import { OutletCard } from '@/components/OutletCard'
import { cn, relativeTime } from '@/lib/utils'

const statusLabel: Record<string, string> = {
  normal: 'Stable',
  warning: 'Warning',
  critical: 'Critical',
}

const statusStyle: Record<string, string> = {
  normal: 'text-secondary bg-secondary/10',
  warning: 'text-amber-400 bg-amber-400/10',
  critical: 'text-tertiary bg-tertiary/10',
}

const initiatedByStyle: Record<string, string> = {
  ui: 'text-primary',
  cli: 'text-on-surface-dim',
  mcp: 'text-secondary',
  api: 'text-on-surface-dim',
  apex: 'text-amber-400',
}

function SkeletonCard() {
  return (
    <div className="bg-surface-container rounded-2xl p-5 animate-pulse">
      <div className="h-3 w-24 bg-surface-container-high rounded mb-4" />
      <div className="h-10 w-32 bg-surface-container-high rounded mb-3" />
      <div className="h-8 w-full bg-surface-container-high rounded" />
    </div>
  )
}

export default function Dashboard() {
  const { data: probeData, isLoading: probesLoading } = useProbes()
  const { data: outletData, isLoading: outletsLoading } = useOutlets()
  const { data: eventsData } = useOutletEvents({ limit: 8 })
  const { data: systemData } = useSystemStatus()

  const probes = probeData?.probes ?? []
  const outlets = outletData?.outlets.filter((o) => o.type === 'outlet') ?? []
  const events = eventsData?.events ?? []

  // Overall status — worst of all probes
  const worstStatus = probes.reduce<string>((worst, p) => {
    if (p.status === 'critical') return 'critical'
    if (p.status === 'warning' && worst !== 'critical') return 'warning'
    return worst
  }, 'normal')

  const activeOutlets = outlets.filter(
    (o) => o.state === 'ON' || o.state === 'AON' || o.state === 'TBL',
  ).length

  const criticalCount = probes.filter((p) => p.status === 'critical').length

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
            Welcome back.
          </h1>
          <p className="text-on-surface-dim mt-1">
            {probes.length > 0 ? (
              <>
                Your abyssal reef ecosystem is thriving. {activeOutlets} modules active
                {criticalCount > 0
                  ? `, ${criticalCount} critical alert${criticalCount > 1 ? 's' : ''}`
                  : ', 0 critical alerts'}
                .
              </>
            ) : (
              'Loading telemetry data...'
            )}
          </p>
        </div>

        <span
          className={cn(
            'px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest whitespace-nowrap self-start',
            statusStyle[worstStatus] ?? statusStyle.normal,
          )}
        >
          {statusLabel[worstStatus] ?? 'Stable'}
        </span>
      </div>

      {/* Probe cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {probesLoading
          ? Array.from({ length: 4 }).map((_, i) => <SkeletonCard key={i} />)
          : probes.map((probe) => (
              <ProbeCard key={probe.name} probe={probe} />
            ))}
      </div>

      {/* Power management + System events */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        {/* Outlets */}
        <div className="lg:col-span-3 space-y-4">
          <h2 className="text-lg font-semibold text-on-surface">
            Power Management
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {outletsLoading
              ? Array.from({ length: 4 }).map((_, i) => <SkeletonCard key={i} />)
              : outlets.map((outlet) => (
                  <OutletCard key={outlet.id} outlet={outlet} />
                ))}
          </div>
          {!outletsLoading && outlets.length === 0 && (
            <div className="bg-surface-container rounded-2xl p-8 text-center">
              <p className="text-on-surface-dim">No outlets detected</p>
            </div>
          )}
        </div>

        {/* System events */}
        <div className="lg:col-span-2 space-y-4">
          <h2 className="text-lg font-semibold text-on-surface">
            System Events
          </h2>
          <div className="bg-surface-container rounded-2xl p-4 space-y-1">
            {events.length === 0 ? (
              <p className="text-sm text-on-surface-dim py-4 text-center">
                No recent events
              </p>
            ) : (
              events.map((event) => (
                <div
                  key={event.id}
                  className="flex items-center justify-between py-2.5"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm text-on-surface truncate">
                        {event.outlet_name}
                      </p>
                      <p className="text-xs text-on-surface-dim">
                        {event.from_state} → {event.to_state}
                      </p>
                    </div>
                  </div>
                  <div className="text-right shrink-0 ml-3">
                    <p className="text-xs text-on-surface-dim">
                      {relativeTime(event.ts)}
                    </p>
                    <p
                      className={cn(
                        'text-xs font-medium uppercase',
                        initiatedByStyle[event.initiated_by] ?? 'text-on-surface-dim',
                      )}
                    >
                      {event.initiated_by}
                    </p>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* System health */}
          {systemData && (
            <div className="bg-surface-container rounded-2xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <span
                  className={cn(
                    'h-2 w-2 rounded-full',
                    systemData.poller.poll_ok
                      ? 'bg-secondary animate-bio-pulse'
                      : 'bg-tertiary',
                  )}
                />
                <span className="text-sm font-medium text-on-surface">
                  {systemData.poller.poll_ok ? 'System Healthy' : 'System Degraded'}
                </span>
              </div>
              <div className="text-xs text-on-surface-dim space-y-1">
                <p>Controller: {systemData.controller.serial}</p>
                <p>Firmware: {systemData.controller.firmware}</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

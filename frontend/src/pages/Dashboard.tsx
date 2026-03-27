import { useState } from 'react'
import { Lock, Unlock } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets, useOutletEvents } from '@/hooks/useOutlets'
import { useDevices } from '@/hooks/useDevices'
import { useSystemStatus } from '@/hooks/useSystem'
import { usePageTitle } from '@/hooks/usePageTitle'
import { useDashboardLayout } from '@/hooks/useDashboardLayout'
import { ProbeCard } from '@/components/ProbeCard'
import { OutletCard } from '@/components/OutletCard'
import { DeviceCard } from '@/components/DeviceCard'
import { cn, relativeTime } from '@/lib/utils'
import type { DashboardItem, Probe, Outlet, Device } from '@/api/types'

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

/** Split dashboard items into sections by separator items. */
function groupIntoSections(items: DashboardItem[]): { label: string | null; items: DashboardItem[] }[] {
  const sections: { label: string | null; items: DashboardItem[] }[] = []
  let current: { label: string | null; items: DashboardItem[] } = { label: null, items: [] }

  for (const item of items) {
    if (item.item_type === 'separator') {
      if (current.items.length > 0 || current.label !== null) {
        sections.push(current)
      }
      current = { label: item.label, items: [] }
    } else {
      current.items.push(item)
    }
  }
  if (current.items.length > 0 || current.label !== null) {
    sections.push(current)
  }

  return sections
}

function renderCard(
  item: DashboardItem,
  probeMap: Map<string, Probe>,
  outletMap: Map<string, Outlet>,
  deviceMap: Map<number, Device>,
  controlsLocked: boolean,
) {
  const ref = item.reference_id
  if (!ref) return null

  switch (item.item_type) {
    case 'probe': {
      const probe = probeMap.get(ref)
      if (!probe) return null
      return <ProbeCard key={`probe:${ref}`} probe={probe} />
    }
    case 'outlet': {
      const outlet = outletMap.get(ref)
      if (!outlet) return null
      return <OutletCard key={`outlet:${ref}`} outlet={outlet} controlsLocked={controlsLocked} />
    }
    case 'device': {
      const device = deviceMap.get(Number(ref))
      if (!device) return null
      return (
        <DeviceCard
          key={`device:${ref}`}
          device={device}
          probes={device.probe_names
            .map((name) => probeMap.get(name))
            .filter((p): p is NonNullable<typeof p> => !!p)}
          outlet={device.outlet_id ? outletMap.get(device.outlet_id) : undefined}
          controlsLocked={controlsLocked}
        />
      )
    }
    default:
      return null
  }
}

export default function Dashboard() {
  usePageTitle('Dashboard')
  const [controlsLocked, setControlsLocked] = useState(false)
  const { data: layoutData, isLoading: layoutLoading } = useDashboardLayout()
  const { data: probeData, isLoading: probesLoading } = useProbes()
  const { data: outletData, isLoading: outletsLoading } = useOutlets()
  const { data: deviceData, isLoading: devicesLoading } = useDevices()
  const { data: eventsData } = useOutletEvents({ limit: 8 })
  const { data: systemData } = useSystemStatus()

  const probes = probeData?.probes ?? []
  const allOutlets = (outletData?.outlets ?? []).filter((o) => o.type === 'outlet' || o.type === 'virtual')
  const events = eventsData?.events ?? []
  const devices = deviceData?.devices ?? []
  const dashboardItems = layoutData?.items ?? []

  const probeMap = new Map(probes.map((p) => [p.name, p]))
  const outletMap = new Map(allOutlets.map((o) => [o.id, o]))
  const deviceMap = new Map(devices.map((d) => [d.id, d]))

  // Overall status — worst of all probes
  const worstStatus = probes.reduce<string>((worst, p) => {
    if (p.status === 'critical') return 'critical'
    if (p.status === 'warning' && worst !== 'critical') return 'warning'
    return worst
  }, 'normal')

  // Count active outlets from dashboard items only
  const dashboardOutletIds = new Set(
    dashboardItems.filter((i) => i.item_type === 'outlet').map((i) => i.reference_id),
  )
  const activeOutlets = allOutlets.filter(
    (o) => dashboardOutletIds.has(o.id) && (o.state === 'ON' || o.state === 'AON' || o.state === 'TBL'),
  ).length

  const criticalCount = probes.filter((p) => p.status === 'critical').length

  const isLoading = layoutLoading || probesLoading || outletsLoading || devicesLoading

  const sections = groupIntoSections(dashboardItems)

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

        <div className="flex items-center gap-3 self-start">
          <button
            onClick={() => setControlsLocked((prev) => !prev)}
            className={cn(
              'flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest transition-fluid',
              controlsLocked
                ? 'bg-tertiary/15 text-tertiary'
                : 'bg-surface-container text-on-surface-dim hover:text-on-surface',
            )}
          >
            {controlsLocked ? <Lock size={12} /> : <Unlock size={12} />}
            {controlsLocked ? 'Controls Locked' : 'Lock Controls'}
          </button>
          <span
            className={cn(
              'px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest whitespace-nowrap',
              statusStyle[worstStatus] ?? statusStyle.normal,
            )}
          >
            {statusLabel[worstStatus] ?? 'Stable'}
          </span>
        </div>
      </div>

      {/* Dashboard sections driven by dashboard_items */}
      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
      ) : dashboardItems.length === 0 ? (
        <div className="bg-surface-container rounded-2xl p-8 text-center">
          <p className="text-on-surface-dim">
            No items on dashboard. Go to Settings → Dashboard to add items.
          </p>
        </div>
      ) : (
        sections.map((section, sectionIdx) => {
          const cards = section.items
            .map((item) => renderCard(item, probeMap, outletMap, deviceMap, controlsLocked))
            .filter(Boolean)

          if (cards.length === 0 && section.label === null) return null

          return (
            <div key={sectionIdx} className="space-y-4">
              {section.label && (
                <h2 className="text-lg font-semibold text-on-surface">
                  {section.label}
                </h2>
              )}
              {cards.length > 0 && (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  {cards}
                </div>
              )}
            </div>
          )
        })
      )}

      {/* System events sidebar — always visible, not part of dashboard items */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        <div className="lg:col-span-3" />
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
                        {event.outlet_display_name ?? event.outlet_name}
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

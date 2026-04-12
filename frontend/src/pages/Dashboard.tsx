import { useState } from 'react'
import { Lock, Unlock } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets } from '@/hooks/useOutlets'
import { useDevices } from '@/hooks/useDevices'
import { useSystemStatus } from '@/hooks/useSystem'
import { useFeedStatus } from '@/hooks/useFeed'
import { usePageTitle } from '@/hooks/usePageTitle'
import { useDashboardLayout } from '@/hooks/useDashboardLayout'
import { ProbeCard } from '@/components/ProbeCard'
import { OutletCard } from '@/components/OutletCard'
import { DeviceCard } from '@/components/DeviceCard'
import { FeedCard } from '@/components/FeedCard'
import { MeasurementCard } from '@/components/MeasurementCard'
import {
  ProbeCompactCard,
  OutletCompactCard,
  MeasurementCompactCard,
  FeedCompactCard,
  DeviceCompactCard,
} from '@/components/CompactCard'
import { cn, relativeTime } from '@/lib/utils'
import type { DashboardItem, Probe, Outlet, Device } from '@/api/types'

// ─── Status configuration ────────────────────────────────────────────────────

const statusConfig = {
  normal: {
    glow: 'rgba(58, 223, 250, 0.12)',
    dotClass: 'bg-secondary animate-bio-pulse',
    headline: 'All clear.',
    detail: (active: number, critical: number) =>
      `${active} outlet${active !== 1 ? 's' : ''} running · ${critical > 0 ? `${critical} critical alert${critical > 1 ? 's' : ''}` : 'no alerts'}`,
  },
  warning: {
    glow: 'rgba(251, 191, 36, 0.12)',
    dotClass: 'bg-amber-400',
    headline: 'Something\'s off.',
    detail: (active: number, _critical: number) =>
      `${active} outlet${active !== 1 ? 's' : ''} running · check probe readings`,
  },
  critical: {
    glow: 'rgba(255, 135, 150, 0.15)',
    dotClass: 'bg-tertiary animate-bio-pulse',
    headline: 'Check your tank.',
    detail: (_active: number, critical: number) =>
      `${critical} critical alert${critical > 1 ? 's' : ''} — immediate attention needed`,
  },
} as const

// ─── Sub-components ───────────────────────────────────────────────────────────

function SkeletonCard() {
  return (
    <div className="bg-surface-container rounded-2xl p-5 animate-pulse">
      <div className="h-3 w-24 bg-surface-container-high rounded mb-4" />
      <div className="h-10 w-32 bg-surface-container-high rounded mb-3" />
      <div className="h-8 w-full bg-surface-container-high rounded" />
    </div>
  )
}

function TankStatusHeader({
  worstStatus,
  activeOutlets,
  criticalCount,
  lastPoll,
  pollOk,
  controlsLocked,
  onToggleLock,
  feedActive,
}: {
  worstStatus: string
  activeOutlets: number
  criticalCount: number
  lastPoll?: string
  pollOk?: boolean
  controlsLocked: boolean
  onToggleLock: () => void
  feedActive: boolean
}) {
  const cfg = statusConfig[worstStatus as keyof typeof statusConfig] ?? statusConfig.normal

  return (
    <div
      className="rounded-2xl px-5 py-6 sm:px-7 mb-8 overflow-hidden relative transition-fluid"
      style={{ backgroundColor: 'var(--color-surface-container-low)' }}
    >
      {/* Ambient glow — fills from left, shifts with status */}
      <div
        className="absolute inset-0 pointer-events-none transition-fluid"
        style={{
          background: `radial-gradient(ellipse 60% 160% at -5% 60%, ${cfg.glow}, transparent 70%)`,
        }}
      />

      <div className="relative flex items-center justify-between gap-3">
        <div className="flex items-center gap-4 min-w-0 flex-1">
          {/* Status orb */}
          <div className="relative shrink-0">
            <span className={cn('h-3 w-3 rounded-full block', cfg.dotClass)} />
          </div>

          <div className="min-w-0">
            <h1 className="text-2xl sm:text-3xl font-bold text-on-surface tracking-tight leading-none mb-1.5">
              {cfg.headline}
            </h1>
            <p className="text-sm text-on-surface-dim leading-snug">
              {cfg.detail(activeOutlets, criticalCount)}
              {feedActive && (
                <span className="ml-2 text-primary font-semibold">· Feed active</span>
              )}
            </p>
            {lastPoll && (
              <p className="text-xs text-on-surface-faint mt-1">
                {pollOk === false ? '⚠ Poller down · ' : ''}{relativeTime(lastPoll)}
              </p>
            )}
          </div>
        </div>

        <button
          onClick={onToggleLock}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold uppercase tracking-widest shrink-0 transition-fluid',
            controlsLocked
              ? 'text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container'
              : 'text-primary bg-primary/10 hover:bg-primary/15',
          )}
          aria-label={controlsLocked ? 'Unlock controls' : 'Lock controls'}
        >
          {controlsLocked ? <Lock size={11} /> : <Unlock size={11} />}
          <span className="hidden sm:inline">
            {controlsLocked ? 'Locked' : 'Unlocked'}
          </span>
        </button>
      </div>
    </div>
  )
}

function ZoneMarker({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2.5 mb-4">
      <span className="h-1.5 w-1.5 rounded-full bg-primary/40 shrink-0" />
      <span className="text-[11px] font-bold text-on-surface-faint uppercase tracking-[0.12em] whitespace-nowrap">
        {label}
      </span>
      <div
        className="flex-1 h-px"
        style={{ background: 'linear-gradient(to right, rgba(58,223,250,0.2), transparent)' }}
      />
    </div>
  )
}

// ─── Card rendering ───────────────────────────────────────────────────────────

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
  const compact = item.display_mode === 'compact'

  if (compact) {
    switch (item.item_type) {
      case 'probe': {
        const probe = probeMap.get(ref ?? '')
        if (!probe) return null
        return (
          <div key={`probe:${ref}`} className="col-span-1 lg:col-span-2">
            <ProbeCompactCard probe={probe} />
          </div>
        )
      }
      case 'outlet': {
        const outlet = outletMap.get(ref ?? '')
        if (!outlet) return null
        return (
          <div key={`outlet:${ref}`} className="col-span-1 lg:col-span-2">
            <OutletCompactCard outlet={outlet} />
          </div>
        )
      }
      case 'device': {
        const device = deviceMap.get(Number(ref))
        if (!device) return null
        const primaryProbe = device.probe_names.map((n) => probeMap.get(n)).find(Boolean)
        return (
          <div key={`device:${ref}`} className="col-span-1 lg:col-span-2">
            <DeviceCompactCard device={device} primaryProbe={primaryProbe} />
          </div>
        )
      }
      case 'feed_mode':
        return (
          <div key="feed_mode" className="col-span-1 lg:col-span-2">
            <FeedCompactCard />
          </div>
        )
      case 'measurement':
        if (!ref) return null
        return (
          <div key={`measurement:${ref}`} className="col-span-1 lg:col-span-2">
            <MeasurementCompactCard parameter={ref} />
          </div>
        )
      default:
        return null
    }
  }

  if (!ref) return null

  switch (item.item_type) {
    case 'probe': {
      const probe = probeMap.get(ref)
      if (!probe) return null
      return (
        <div key={`probe:${ref}`} className="col-span-2">
          <ProbeCard probe={probe} />
        </div>
      )
    }
    case 'outlet': {
      const outlet = outletMap.get(ref)
      if (!outlet) return null
      return (
        <div key={`outlet:${ref}`} className="col-span-2">
          <OutletCard outlet={outlet} controlsLocked={controlsLocked} />
        </div>
      )
    }
    case 'device': {
      const device = deviceMap.get(Number(ref))
      if (!device) return null
      return (
        <div key={`device:${ref}`} className="col-span-2">
          <DeviceCard
            device={device}
            probes={device.probe_names
              .map((name) => probeMap.get(name))
              .filter((p): p is NonNullable<typeof p> => !!p)}
            outlet={device.outlet_id ? outletMap.get(device.outlet_id) : undefined}
            controlsLocked={controlsLocked}
          />
        </div>
      )
    }
    case 'feed_mode':
      return (
        <div key="feed_mode" className="col-span-2">
          <FeedCard controlsLocked={controlsLocked} />
        </div>
      )
    case 'measurement':
      return (
        <div key={`measurement:${ref}`} className="col-span-2">
          <MeasurementCard parameter={ref} />
        </div>
      )
    default:
      return null
  }
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Dashboard() {
  usePageTitle('Dashboard')
  const [controlsLocked, setControlsLocked] = useState(true)
  const { data: layoutData, isLoading: layoutLoading } = useDashboardLayout()
  const { data: probeData, isLoading: probesLoading } = useProbes()
  const { data: outletData, isLoading: outletsLoading } = useOutlets()
  const { data: deviceData, isLoading: devicesLoading } = useDevices()
  const { data: systemData } = useSystemStatus()
  const { data: feedData } = useFeedStatus()

  const probes = probeData?.probes ?? []
  const allOutlets = (outletData?.outlets ?? []).filter(
    (o) => o.type === 'outlet' || o.type === 'virtual',
  )
  const devices = deviceData?.devices ?? []
  const dashboardItems = layoutData?.items ?? []

  const probeMap = new Map(probes.map((p) => [p.name, p]))
  const outletMap = new Map(allOutlets.map((o) => [o.id, o]))
  const deviceMap = new Map(devices.map((d) => [d.id, d]))

  const worstStatus = probes.reduce<string>((worst, p) => {
    if (p.status === 'critical') return 'critical'
    if (p.status === 'warning' && worst !== 'critical') return 'warning'
    return worst
  }, 'normal')

  const activeOutlets = allOutlets.filter(
    (o) => o.state === 'ON' || o.state === 'AON' || o.state === 'TBL',
  ).length

  const criticalCount = probes.filter((p) => p.status === 'critical').length
  const feedActive = (feedData?.active ?? 0) === 1
  const isLoading = layoutLoading || probesLoading || outletsLoading || devicesLoading
  const sections = groupIntoSections(dashboardItems)

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto">
      <TankStatusHeader
        worstStatus={worstStatus}
        activeOutlets={activeOutlets}
        criticalCount={criticalCount}
        lastPoll={systemData?.poller.last_poll_ts}
        pollOk={systemData?.poller.poll_ok}
        controlsLocked={controlsLocked}
        onToggleLock={() => setControlsLocked((v) => !v)}
        feedActive={feedActive}
      />

      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-8 gap-3 items-start" style={{ gridAutoFlow: 'dense' }}>
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="col-span-2">
              <SkeletonCard />
            </div>
          ))}
        </div>
      ) : dashboardItems.length === 0 ? (
        <div className="bg-surface-container rounded-2xl p-10 text-center">
          <p className="text-on-surface-dim mb-2">Your dashboard is empty.</p>
          <p className="text-sm text-on-surface-faint">
            Go to{' '}
            <a href="/settings" className="text-primary hover:underline">
              Settings → Dashboard
            </a>{' '}
            to add probes, outlets, and more.
          </p>
        </div>
      ) : (
        <div className="space-y-8">
          {sections.map((section, sectionIdx) => {
            const cards = section.items
              .map((item) =>
                renderCard(item, probeMap, outletMap, deviceMap, controlsLocked),
              )
              .filter(Boolean)

            if (cards.length === 0 && section.label === null) return null

            return (
              <div key={sectionIdx}>
                {section.label && <ZoneMarker label={section.label} />}
                {cards.length > 0 && (
                  <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-8 gap-3 items-start" style={{ gridAutoFlow: 'dense' }}>
                    {cards}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

    </div>
  )
}

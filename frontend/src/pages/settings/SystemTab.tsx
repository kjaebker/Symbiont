import { cn,relativeTime,formatBytes } from '@/lib/utils'
import { getBubblesEnabled,setBubblesEnabled } from '@/api/client'
import { RefreshCw,Activity,Droplets } from 'lucide-react'
import { useSystemStatus } from '@/hooks/useSystem'
import { useEventBusStats } from '@/hooks/useEvents'
import { Chip, IconButton, Toggle, LoadingState, EmptyState, labelClass } from '@/components/ui'

// =============================================================================
// System Tab
// =============================================================================

function EventBusPanel() {
  const { data, isLoading } = useEventBusStats()
  const subs = data?.subscribers ?? []

  if (isLoading) return <LoadingState label="Loading bus stats..." />

  return (
    <div>
      <p className={cn(labelClass, 'px-4 mb-1')}>Event Bus</p>
      <div className="bg-surface-container-high/40 rounded-xl overflow-hidden">
        {subs.length === 0 && (
          <p className="px-4 py-3 text-sm text-on-surface-faint">No subscribers registered.</p>
        )}
        {subs.map((s) => (
          <div key={s.name} className="px-4 py-3">
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-on-surface font-medium font-mono">{s.name}</span>
              {s.dropped > 0 && (
                <Chip tone="critical">{s.dropped} dropped</Chip>
              )}
              {s.dropped === 0 && (
                <Chip tone="secondary">healthy</Chip>
              )}
            </div>
            <div className="flex gap-4">
              <div>
                <p className="text-xs text-on-surface-faint">Queue</p>
                <p className="text-sm text-on-surface font-mono">{s.queue_depth}</p>
              </div>
              <div>
                <p className="text-xs text-on-surface-faint">Published</p>
                <p className="text-sm text-on-surface font-mono">{s.published.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-xs text-on-surface-faint">Dropped</p>
                <p className={cn('text-sm font-mono', s.dropped > 0 ? 'text-tertiary' : 'text-on-surface')}>{s.dropped}</p>
              </div>
            </div>
            {s.last_error && (
              <p className="text-xs text-tertiary mt-1 truncate" title={s.last_error}>{s.last_error}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export default function SystemTab() {
  const { data, isLoading, refetch, isFetching } = useSystemStatus()

  if (isLoading) return <LoadingState label="Loading system status..." />

  if (!data) {
    return (
      <EmptyState
        icon={<Activity size={32} />}
        message="System status unavailable. The API server may not be reachable."
      />
    )
  }

  const stat = (label: string, value: string) => (
    <div className="py-3 px-4 flex items-center justify-between">
      <span className={labelClass}>{label}</span>
      <span className="text-sm text-on-surface font-mono">{value}</span>
    </div>
  )

  return (
    <div className="space-y-6 p-4">
      {/* Poller status banner */}
      <div
        className={cn(
          'flex items-center gap-3 rounded-2xl px-4 py-3',
          data.poller.poll_ok ? 'bg-secondary/10' : 'bg-tertiary/10',
        )}
      >
        <span
          className={cn(
            'h-2.5 w-2.5 rounded-full shrink-0',
            data.poller.poll_ok ? 'bg-secondary animate-bio-pulse' : 'bg-tertiary',
          )}
        />
        <div className="flex-1">
          <p className={cn('text-sm font-semibold', data.poller.poll_ok ? 'text-secondary' : 'text-tertiary')}>
            {data.poller.poll_ok ? 'Poller running' : 'Poller degraded'}
          </p>
          <p className="text-xs text-on-surface-dim mt-0.5">
            Last poll {relativeTime(data.poller.last_poll_ts)} · every {data.poller.poll_interval_seconds}s
          </p>
        </div>
        <IconButton
          icon={RefreshCw}
          label="Refresh status"
          onClick={() => refetch()}
          disabled={isFetching}
          className={isFetching ? '[&_svg]:animate-spin' : undefined}
        />
      </div>

      {/* Controller */}
      <div>
        <p className={cn(labelClass, 'px-4 mb-1')}>Controller</p>
        <div className="bg-surface-container-high/40 rounded-xl py-1">
          {stat('Serial', data.controller.serial)}
          {stat('Firmware', data.controller.firmware)}
          {stat('Hardware', data.controller.hardware)}
        </div>
      </div>

      {/* Databases */}
      <div>
        <p className={cn(labelClass, 'px-4 mb-1')}>Databases</p>
        <div className="bg-surface-container-high/40 rounded-xl py-1">
          {stat('Telemetry (DuckDB)', formatBytes(data.db.duckdb_size_bytes))}
          {stat('App state (SQLite)', formatBytes(data.db.sqlite_size_bytes))}
        </div>
      </div>

      {/* Event Bus */}
      <EventBusPanel />

      {/* Interface */}
      <div>
        <p className={cn(labelClass, 'px-4 mb-1')}>Interface</p>
        <div className="bg-surface-container-high/40 rounded-xl">
          <div className="py-3 px-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Droplets size={16} className="text-primary/60" />
              <div>
                <p className="text-sm text-on-surface font-medium">Floating bubbles</p>
                <p className="text-xs text-on-surface-faint">Decorative background animation</p>
              </div>
            </div>
            <Toggle
              checked={getBubblesEnabled()}
              label="Floating bubbles"
              onChange={(v) => { setBubblesEnabled(v); window.location.reload() }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}


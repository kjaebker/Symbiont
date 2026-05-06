import { cn,relativeTime,formatBytes } from '@/lib/utils'
import { getBubblesEnabled,setBubblesEnabled } from '@/api/client'
import { RefreshCw,Activity,Droplets } from 'lucide-react'
import { useSystemStatus } from '@/hooks/useSystem'
import { useEventBusStats } from '@/hooks/useEvents'
import { LoadingState,EmptyState } from './_shared'

// =============================================================================
// System Tab
// =============================================================================

function EventBusPanel() {
  const { data, isLoading } = useEventBusStats()
  const subs = data?.subscribers ?? []

  if (isLoading) return <LoadingState label="Loading bus stats..." />

  return (
    <div>
      <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Event Bus</p>
      <div className="bg-surface-container-high/40 rounded-xl overflow-hidden">
        {subs.length === 0 && (
          <p className="px-4 py-3 text-sm text-on-surface-faint">No subscribers registered.</p>
        )}
        {subs.map((s, i) => (
          <div
            key={s.name}
            className={cn(
              'px-4 py-3',
              i < subs.length - 1 ? 'border-b border-surface-container-highest/30' : '',
            )}
          >
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-on-surface font-medium font-mono">{s.name}</span>
              {s.dropped > 0 && (
                <span className="text-xs px-1.5 py-0.5 rounded-full bg-tertiary/10 text-tertiary font-medium">
                  {s.dropped} dropped
                </span>
              )}
              {s.dropped === 0 && (
                <span className="text-xs px-1.5 py-0.5 rounded-full bg-secondary/10 text-secondary font-medium">
                  healthy
                </span>
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
      <span className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">{label}</span>
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
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="p-1.5 rounded-lg text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high transition-fluid cursor-pointer disabled:opacity-50"
        >
          <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
        </button>
      </div>

      {/* Controller */}
      <div>
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Controller</p>
        <div className="bg-surface-container-high/40 rounded-xl divide-y divide-surface-container-highest/50">
          {stat('Serial', data.controller.serial)}
          {stat('Firmware', data.controller.firmware)}
          {stat('Hardware', data.controller.hardware)}
        </div>
      </div>

      {/* Databases */}
      <div>
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Databases</p>
        <div className="bg-surface-container-high/40 rounded-xl divide-y divide-surface-container-highest/50">
          {stat('Telemetry (DuckDB)', formatBytes(data.db.duckdb_size_bytes))}
          {stat('App state (SQLite)', formatBytes(data.db.sqlite_size_bytes))}
        </div>
      </div>

      {/* Event Bus */}
      <EventBusPanel />

      {/* Interface */}
      <div>
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Interface</p>
        <div className="bg-surface-container-high/40 rounded-xl">
          <div className="py-3 px-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Droplets size={16} className="text-primary/60" />
              <div>
                <p className="text-sm text-on-surface font-medium">Floating bubbles</p>
                <p className="text-xs text-on-surface-faint">Decorative background animation</p>
              </div>
            </div>
            <button
              onClick={() => { setBubblesEnabled(!getBubblesEnabled()); window.location.reload() }}
              className={cn(
                'px-3 py-1 rounded-full text-xs font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
                getBubblesEnabled()
                  ? 'bg-primary/20 text-primary'
                  : 'bg-surface-container-highest text-on-surface-faint',
              )}
            >
              {getBubblesEnabled() ? 'On' : 'Off'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}


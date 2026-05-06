import { useState } from 'react'
import { cn } from '@/lib/utils'
import { RefreshCw,Terminal } from 'lucide-react'
import { useSystemLog } from '@/hooks/useSystem'
import type { SystemLogLine } from '@/api/types'
import { LoadingState,EmptyState } from './_shared'

// =============================================================================
// System Log Tab
// =============================================================================

const levelBadge: Record<string, string> = {
  ERROR: 'bg-tertiary/15 text-tertiary',
  WARN:  'bg-amber-400/15 text-amber-400',
  INFO:  'bg-primary/10 text-primary',
  DEBUG: 'bg-surface-container-highest text-on-surface-faint',
}

const levelText: Record<string, string> = {
  ERROR: 'text-tertiary',
  WARN:  'text-amber-400',
  INFO:  'text-on-surface-dim',
  DEBUG: 'text-on-surface-faint',
}

const serviceBadge: Record<string, string> = {
  api:    'bg-primary/10 text-primary',
  poller: 'bg-secondary/10 text-secondary',
}

function LogEntry({ line }: { line: SystemLogLine }) {
  const [expanded, setExpanded] = useState(false)
  const hasFields = line.fields && Object.keys(line.fields).length > 0

  return (
    <div
      className={cn(
        'px-4 py-1.5 hover:bg-surface-container-high/50 transition-fluid',
        hasFields && 'cursor-pointer',
      )}
      onClick={() => hasFields && setExpanded(!expanded)}
    >
      <div className="flex items-baseline gap-2 flex-wrap">
        <span className="text-on-surface-faint shrink-0 tabular-nums">
          {line.ts ? new Date(line.ts).toLocaleTimeString() : '—'}
        </span>
        <span className={cn('shrink-0 px-1.5 py-0.5 rounded text-xs font-medium uppercase', serviceBadge[line.service] ?? 'bg-surface-container-highest text-on-surface-faint')}>
          {line.service}
        </span>
        <span className={cn('shrink-0 px-1.5 py-0.5 rounded text-xs font-medium uppercase', levelBadge[line.level] ?? levelBadge.INFO)}>
          {line.level}
        </span>
        <span className={cn('flex-1 min-w-0', levelText[line.level] ?? levelText.INFO)}>
          {line.msg}
        </span>
      </div>
      {expanded && hasFields && (
        <div className="mt-1 pl-4 space-y-0.5">
          {Object.entries(line.fields!).map(([k, v]) => (
            <div key={k} className="flex gap-1">
              <span className="text-on-surface-dim">{k}</span>
              <span className="text-on-surface-faint">=</span>
              <span className="text-on-surface-dim">{String(v)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default function SystemLogTab() {
  const [service, setService] = useState('')
  const { data, isLoading, refetch, isFetching } = useSystemLog(
    service ? { service, limit: 200 } : { limit: 200 },
  )

  const lines = data?.lines ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-4 pt-2">
        <div className="flex items-center gap-3">
          <span className="text-xs text-on-surface-faint uppercase tracking-widest">System Log</span>
          <div className="flex gap-1">
            {(['', 'api', 'poller'] as const).map((s) => (
              <button
                key={s}
                onClick={() => setService(s)}
                className={cn(
                  'px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider transition-fluid cursor-pointer',
                  service === s
                    ? 'bg-primary/20 text-primary'
                    : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                )}
              >
                {s || 'All'}
              </button>
            ))}
          </div>
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer disabled:opacity-50"
        >
          <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {isLoading ? (
        <LoadingState label="Loading logs..." />
      ) : lines.length === 0 ? (
        <EmptyState
          icon={<Terminal size={32} />}
          message="No log entries found. Logs are read from the systemd journal when running as a service."
        />
      ) : (
        <div className="overflow-y-auto max-h-[600px] font-mono text-xs">
          {[...lines].reverse().map((line, i) => (
            <LogEntry key={i} line={line} />
          ))}
        </div>
      )}
    </div>
  )
}


import { useState, useEffect, useMemo, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useProbeHistory } from '@/hooks/useProbes'
import { ProbeChart } from '@/components/ProbeChart'
import { TimeRangePicker } from '@/components/TimeRangePicker'
import { ProbeSelector, SERIES_COLORS } from '@/components/ProbeSelector'
import { cn } from '@/lib/utils'

const INTERVALS = [
  { label: 'Auto', value: '' },
  { label: '10s', value: '10s' },
  { label: '1m', value: '1m' },
  { label: '5m', value: '5m' },
  { label: '15m', value: '15m' },
  { label: '1h', value: '1h' },
  { label: '1d', value: '1d' },
]

function parseRange(searchParams: URLSearchParams): {
  from: Date
  to: Date
} {
  const fromParam = searchParams.get('from')
  const toParam = searchParams.get('to')
  const now = new Date()
  return {
    from: fromParam ? new Date(fromParam) : new Date(now.getTime() - 24 * 60 * 60 * 1000),
    to: toParam ? new Date(toParam) : now,
  }
}

export default function History() {
  const [searchParams, setSearchParams] = useSearchParams()

  // Parse URL state
  const probeParam = searchParams.get('probe')
  const initialProbes = probeParam ? probeParam.split(',').filter(Boolean) : []
  const intervalParam = searchParams.get('interval') ?? ''

  const [selectedProbes, setSelectedProbes] = useState<string[]>(initialProbes)
  const [range, setRange] = useState(() => parseRange(searchParams))
  const [interval, setInterval] = useState(intervalParam)

  // Sync state to URL
  const syncUrl = useCallback(
    (probes: string[], r: { from: Date; to: Date }, int: string) => {
      const params = new URLSearchParams()
      if (probes.length > 0) params.set('probe', probes.join(','))
      params.set('from', r.from.toISOString())
      params.set('to', r.to.toISOString())
      if (int) params.set('interval', int)
      setSearchParams(params, { replace: true })
    },
    [setSearchParams],
  )

  // Sync on changes
  useEffect(() => {
    syncUrl(selectedProbes, range, interval)
  }, [selectedProbes, range, interval, syncUrl])

  const historyParams = useMemo(
    () => ({
      from: range.from.toISOString(),
      to: range.to.toISOString(),
      interval: interval || undefined,
    }),
    [range, interval],
  )

  // Fetch history for each selected probe
  const probe0 = useProbeHistory(selectedProbes[0] ?? null, historyParams)
  const probe1 = useProbeHistory(selectedProbes[1] ?? null, historyParams)
  const probe2 = useProbeHistory(selectedProbes[2] ?? null, historyParams)
  const probe3 = useProbeHistory(selectedProbes[3] ?? null, historyParams)
  const queries = [probe0, probe1, probe2, probe3]

  const chartSeries = useMemo(() => {
    return selectedProbes
      .map((_name, i) => {
        const q = queries[i]
        if (!q?.data) return null
        return {
          name: q.data.probe,
          data: q.data.data,
          unit: '',
          color: SERIES_COLORS[i % SERIES_COLORS.length],
        }
      })
      .filter((s): s is NonNullable<typeof s> => s !== null)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProbes, probe0.data, probe1.data, probe2.data, probe3.data])

  // Summary stats per series
  const stats = useMemo(() => {
    return chartSeries.map((s) => {
      const vals = s.data.map((d) => d.value)
      if (vals.length === 0) return { name: s.name, color: s.color, min: null, max: null, avg: null }
      const min = Math.min(...vals)
      const max = Math.max(...vals)
      const avg = vals.reduce((a, b) => a + b, 0) / vals.length
      return { name: s.name, color: s.color, min, max, avg }
    })
  }, [chartSeries])

  const isLoading = selectedProbes.some((_, i) => queries[i]?.isLoading)

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <p className="text-xs text-primary uppercase tracking-widest mb-2">
          Telemetry Archive
        </p>
        <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
          Aquatic History
        </h1>
      </div>

      {/* Controls */}
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <ProbeSelector
          selected={selectedProbes}
          onChange={setSelectedProbes}
        />

        <div className="flex flex-col gap-3 md:items-end">
          <TimeRangePicker value={range} onChange={setRange} />

          <div className="flex items-center gap-1.5">
            <span className="text-xs text-on-surface-faint uppercase tracking-wider mr-1">
              Interval
            </span>
            {INTERVALS.map((int) => (
              <button
                key={int.value}
                onClick={() => setInterval(int.value)}
                className={cn(
                  'px-2.5 py-1 rounded-full text-xs font-medium transition-fluid',
                  interval === int.value
                    ? 'bg-primary/20 text-primary'
                    : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high',
                )}
              >
                {int.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Chart */}
      {selectedProbes.length === 0 ? (
        <div className="bg-surface-container rounded-2xl p-12 flex flex-col items-center justify-center gap-3">
          <span className="text-on-surface-faint text-sm">
            Select a probe to view its history
          </span>
        </div>
      ) : (
        <ProbeChart series={isLoading ? [] : chartSeries} />
      )}

      {/* Stats */}
      {stats.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {stats.map((s) => (
            <div
              key={s.name}
              className="bg-surface-container rounded-2xl p-4"
            >
              <div className="flex items-center gap-2 mb-3">
                <span
                  className="h-2.5 w-2.5 rounded-full flex-shrink-0"
                  style={{ backgroundColor: s.color }}
                />
                <span className="text-xs text-on-surface-dim uppercase tracking-widest font-medium">
                  {s.name}
                </span>
              </div>
              <div className="grid grid-cols-3 gap-2 text-center">
                <div>
                  <span className="text-xs text-on-surface-faint uppercase tracking-wider block mb-1">
                    Min
                  </span>
                  <span className="text-lg font-bold text-on-surface">
                    {s.min?.toFixed(2) ?? '--'}
                  </span>
                </div>
                <div>
                  <span className="text-xs text-on-surface-faint uppercase tracking-wider block mb-1">
                    Avg
                  </span>
                  <span className="text-lg font-bold text-on-surface">
                    {s.avg?.toFixed(2) ?? '--'}
                  </span>
                </div>
                <div>
                  <span className="text-xs text-on-surface-faint uppercase tracking-wider block mb-1">
                    Max
                  </span>
                  <span className="text-lg font-bold text-on-surface">
                    {s.max?.toFixed(2) ?? '--'}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

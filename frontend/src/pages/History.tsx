import { useState, useEffect, useMemo, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Download } from 'lucide-react'
import { useProbes, useProbeHistory } from '@/hooks/useProbes'
import { useMeasurements } from '@/hooks/useMeasurements'
import { usePageTitle } from '@/hooks/usePageTitle'
import { ProbeChart } from '@/components/ProbeChart'
import { TimeRangePicker } from '@/components/TimeRangePicker'
import { ProbeSelector, MeasurementSelector, SERIES_COLORS } from '@/components/ProbeSelector'
import { getToken } from '@/api/client'
import { FilterDropdown } from '@/components/FilterDropdown'

export const INTERVALS = [
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

export type HistoryRange = { from: Date; to: Date }

interface HistoryProps {
  hideHeader?: boolean
  selectedProbes?: string[]
  onSelectedProbesChange?: (v: string[]) => void
  selectedMeas?: string[]
  onSelectedMeasChange?: (v: string[]) => void
  range?: HistoryRange
  onRangeChange?: (v: HistoryRange) => void
  interval?: string
  onIntervalChange?: (v: string) => void
}

export default function History({
  hideHeader = false,
  selectedProbes: selectedProbesProp,
  onSelectedProbesChange,
  selectedMeas: selectedMeasProp,
  onSelectedMeasChange,
  range: rangeProp,
  onRangeChange,
  interval: intervalProp,
  onIntervalChange,
}: HistoryProps) {
  usePageTitle('History')
  const [searchParams, setSearchParams] = useSearchParams()

  const controlled = hideHeader && onSelectedProbesChange !== undefined

  // Parse URL state for standalone mode
  const probeParam = searchParams.get('probe')
  const initialProbes = probeParam ? probeParam.split(',').filter(Boolean) : []
  const measParam = searchParams.get('meas')
  const initialMeas = measParam ? measParam.split(',').filter(Boolean) : []
  const intervalParam = searchParams.get('interval') ?? ''

  const [selectedProbesInternal, setSelectedProbesInternal] = useState<string[]>(initialProbes)
  const [selectedMeasInternal, setSelectedMeasInternal] = useState<string[]>(initialMeas)
  const [rangeInternal, setRangeInternal] = useState(() => parseRange(searchParams))
  const [intervalInternal, setIntervalInternal] = useState(intervalParam)

  const selectedProbes = controlled ? (selectedProbesProp ?? []) : selectedProbesInternal
  const setSelectedProbes = controlled ? onSelectedProbesChange! : setSelectedProbesInternal
  const selectedMeas = controlled ? (selectedMeasProp ?? []) : selectedMeasInternal
  const setSelectedMeas = controlled ? onSelectedMeasChange! : setSelectedMeasInternal
  const range = controlled ? (rangeProp ?? { from: new Date(Date.now() - 86400000), to: new Date() }) : rangeInternal
  const setRange = controlled ? onRangeChange! : setRangeInternal
  const interval = controlled ? (intervalProp ?? '') : intervalInternal
  const setInterval = controlled ? onIntervalChange! : setIntervalInternal

  // Sync state to URL
  const syncUrl = useCallback(
    (probes: string[], meas: string[], r: { from: Date; to: Date }, int: string) => {
      const params = new URLSearchParams()
      if (probes.length > 0) params.set('probe', probes.join(','))
      if (meas.length > 0) params.set('meas', meas.join(','))
      params.set('from', r.from.toISOString())
      params.set('to', r.to.toISOString())
      if (int) params.set('interval', int)
      setSearchParams(params, { replace: true })
    },
    [setSearchParams],
  )

  // Sync on changes — skip when embedded (parent owns the URL)
  useEffect(() => {
    if (hideHeader) return
    syncUrl(selectedProbes, selectedMeas, range, interval)
  }, [selectedProbes, selectedMeas, range, interval, syncUrl, hideHeader])

  const historyParams = useMemo(
    () => ({
      from: range.from.toISOString(),
      to: range.to.toISOString(),
      interval: interval || undefined,
    }),
    [range, interval],
  )

  // Probe metadata for display names and units
  const { data: probesData } = useProbes()
  const probeMap = useMemo(() => {
    const map = new Map<string, { display_name: string; unit: string }>()
    for (const p of probesData?.probes ?? []) {
      map.set(p.name, { display_name: p.display_name, unit: p.unit })
    }
    return map
  }, [probesData])

  // Fetch history for each selected probe
  const probe0 = useProbeHistory(selectedProbes[0] ?? null, historyParams)
  const probe1 = useProbeHistory(selectedProbes[1] ?? null, historyParams)
  const probe2 = useProbeHistory(selectedProbes[2] ?? null, historyParams)
  const probe3 = useProbeHistory(selectedProbes[3] ?? null, historyParams)
  const queries = [probe0, probe1, probe2, probe3]

  // Fetch measurements for each selected parameter
  const measRangeParams = useMemo(
    () => ({ from: range.from.toISOString(), to: range.to.toISOString() }),
    [range],
  )
  const meas0 = useMeasurements(selectedMeas[0] ? { parameter: selectedMeas[0], ...measRangeParams } : undefined)
  const meas1 = useMeasurements(selectedMeas[1] ? { parameter: selectedMeas[1], ...measRangeParams } : undefined)
  const meas2 = useMeasurements(selectedMeas[2] ? { parameter: selectedMeas[2], ...measRangeParams } : undefined)
  const meas3 = useMeasurements(selectedMeas[3] ? { parameter: selectedMeas[3], ...measRangeParams } : undefined)
  const measQueries = [meas0, meas1, meas2, meas3]

  const chartSeries = useMemo(() => {
    const probeSeries = selectedProbes
      .map((_name, i) => {
        const q = queries[i]
        if (!q?.data) return null
        const meta = probeMap.get(q.data.probe)
        const label = meta?.display_name ?? q.data.probe
        const unit = meta?.unit ?? ''
        return {
          name: unit ? `${label} (${unit})` : label,
          data: q.data.data,
          unit,
          color: SERIES_COLORS[i % SERIES_COLORS.length],
        }
      })
      .filter((s): s is NonNullable<typeof s> => s !== null)

    const measSeries = selectedMeas
      .map((paramName, i) => {
        const q = measQueries[i]
        if (!q?.data) return null
        const measurements = q.data.measurements
        if (measurements.length === 0) return null
        const unit = measurements[0].canonical_unit ?? ''
        return {
          name: unit ? `${paramName} (${unit}) ●` : `${paramName} ●`,
          data: measurements.map((m) => ({ ts: m.measured_at, value: m.value })),
          unit,
          color: SERIES_COLORS[(selectedProbes.length + i) % SERIES_COLORS.length],
          scatter: true as const,
        }
      })
      .filter((s): s is NonNullable<typeof s> => s !== null)

    return [...probeSeries, ...measSeries]
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProbes, selectedMeas, probe0.data, probe1.data, probe2.data, probe3.data, meas0.data, meas1.data, meas2.data, meas3.data, probeMap])

  // Summary stats per series
  const stats = useMemo(() => {
    return chartSeries.map((s) => {
      const vals = s.data.map((d) => d.value)
      if (vals.length === 0) return { name: s.name, color: s.color, unit: s.unit, min: null, max: null, avg: null }
      const min = Math.min(...vals)
      const max = Math.max(...vals)
      const avg = vals.reduce((a, b) => a + b, 0) / vals.length
      return { name: s.name, color: s.color, unit: s.unit, min, max, avg }
    })
  }, [chartSeries])

  const isLoading =
    selectedProbes.some((_, i) => queries[i]?.isLoading) ||
    selectedMeas.some((_, i) => measQueries[i]?.isLoading)

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      {!hideHeader && (
        <div>
          <p className="text-xs text-primary uppercase tracking-widest mb-2">
            Telemetry Archive
          </p>
          <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
            Aquatic History
          </h1>
        </div>
      )}

      {/* Controls — hidden when parent owns them via filterBar */}
      {!controlled && (
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
            <ProbeSelector
              selected={selectedProbes}
              onChange={setSelectedProbes}
            />
            <MeasurementSelector
              selected={selectedMeas}
              onChange={setSelectedMeas}
              colorOffset={selectedProbes.length}
            />
          </div>

          <div className="flex flex-col gap-3 sm:items-end">
            <TimeRangePicker value={range} onChange={setRange} />

            <FilterDropdown
              label="Interval"
              value={interval}
              options={INTERVALS.map((int) => ({ value: int.value, label: int.label }))}
              onChange={setInterval}
            />
          </div>
        </div>
      )}

      {/* Chart */}
      {selectedProbes.length === 0 && selectedMeas.length === 0 ? (
        <div className="bg-surface-container rounded-2xl p-12 flex flex-col items-center justify-center gap-3">
          <span className="text-on-surface-faint text-sm">
            Select a probe or test parameter to view history
          </span>
        </div>
      ) : (
        <ProbeChart series={isLoading ? [] : chartSeries} />
      )}

      {/* Export */}
      {selectedProbes.length > 0 && (
        <div className="flex items-center gap-2">
          {selectedProbes.map((name) => (
            <a
              key={name}
              href={buildExportUrl(name, range)}
              download
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium text-on-surface-dim bg-surface-container hover:bg-surface-container-high transition-fluid"
            >
              <Download size={12} />
              Export {probeMap.get(name)?.display_name ?? name}
            </a>
          ))}
          {selectedProbes.length > 1 && (
            <a
              href={buildBulkExportUrl(range)}
              download
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium text-primary bg-primary/10 hover:bg-primary/20 transition-fluid"
            >
              <Download size={12} />
              Export All (ZIP)
            </a>
          )}
        </div>
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
                    {s.min != null ? (s.min % 1 === 0 ? s.min.toFixed(0) : s.min.toFixed(2)) : '--'}
                    {s.min != null && s.unit && <span className="text-xs text-on-surface-faint ml-0.5">{s.unit}</span>}
                  </span>
                </div>
                <div>
                  <span className="text-xs text-on-surface-faint uppercase tracking-wider block mb-1">
                    Avg
                  </span>
                  <span className="text-lg font-bold text-on-surface">
                    {s.avg != null ? (s.avg % 1 === 0 ? s.avg.toFixed(0) : s.avg.toFixed(2)) : '--'}
                    {s.avg != null && s.unit && <span className="text-xs text-on-surface-faint ml-0.5">{s.unit}</span>}
                  </span>
                </div>
                <div>
                  <span className="text-xs text-on-surface-faint uppercase tracking-wider block mb-1">
                    Max
                  </span>
                  <span className="text-lg font-bold text-on-surface">
                    {s.max != null ? (s.max % 1 === 0 ? s.max.toFixed(0) : s.max.toFixed(2)) : '--'}
                    {s.max != null && s.unit && <span className="text-xs text-on-surface-faint ml-0.5">{s.unit}</span>}
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

function buildExportUrl(name: string, range: { from: Date; to: Date }): string {
  const params = new URLSearchParams()
  params.set('from', range.from.toISOString())
  params.set('to', range.to.toISOString())
  const token = getToken()
  if (token) params.set('token', token)
  return `/api/probes/${encodeURIComponent(name)}/export?${params}`
}

function buildBulkExportUrl(range: { from: Date; to: Date }): string {
  const params = new URLSearchParams()
  params.set('from', range.from.toISOString())
  params.set('to', range.to.toISOString())
  const token = getToken()
  if (token) params.set('token', token)
  return `/api/export?${params}`
}

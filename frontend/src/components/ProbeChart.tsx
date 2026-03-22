import { useEffect, useRef, useState } from 'react'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import { cn } from '@/lib/utils'

interface Series {
  name: string
  data: { ts: string; value: number }[]
  unit: string
  color: string
}

interface ProbeChartProps {
  series: Series[]
  height?: number
}

function buildUPlotData(series: Series[]): uPlot.AlignedData {
  if (series.length === 0 || series[0].data.length === 0) {
    return [[]]
  }

  // Use the first series' timestamps as the x-axis
  const timestamps = series[0].data.map((d) => Math.floor(new Date(d.ts).getTime() / 1000))
  const values = series.map((s) => {
    // Build a map from timestamp to value for this series
    const tsMap = new Map(s.data.map((d) => [Math.floor(new Date(d.ts).getTime() / 1000), d.value]))
    return timestamps.map((t) => tsMap.get(t) ?? null) as (number | null)[]
  })

  return [timestamps, ...values]
}

function buildOpts(
  series: Series[],
  width: number,
  height: number,
): uPlot.Options {
  const cssVar = (name: string) =>
    getComputedStyle(document.documentElement).getPropertyValue(name).trim()

  const gridColor = cssVar('--color-surface-container-high') || '#171f36'
  const textColor = cssVar('--color-on-surface-dim') || '#8a90a8'

  // Each series gets its own scale and axis so different value ranges
  // don't squash each other. Odd series on the left, even on the right.
  const yAxes: uPlot.Axis[] = series.map((s, i) => ({
    stroke: s.color,
    grid: { stroke: i === 0 ? gridColor : 'transparent', width: 1 },
    ticks: { stroke: gridColor, width: 1 },
    font: '11px Manrope, system-ui, sans-serif',
    gap: 8,
    size: 60,
    scale: `y${i}`,
    side: i % 2 === 0 ? 3 : 1, // 3 = left, 1 = right
  }))

  const scales: uPlot.Scales = {
    x: { time: true },
  }
  series.forEach((_s, i) => {
    scales[`y${i}`] = { auto: true }
  })

  return {
    width,
    height,
    cursor: {
      drag: { x: true, y: false },
    },
    axes: [
      {
        stroke: textColor,
        grid: { stroke: gridColor, width: 1 },
        ticks: { stroke: gridColor, width: 1 },
        font: '11px Manrope, system-ui, sans-serif',
        gap: 8,
      },
      ...yAxes,
    ],
    series: [
      {},
      ...series.map((s, i) => ({
        label: s.name,
        stroke: s.color,
        scale: `y${i}`,
        width: 2,
        fill: s.color + '15',
        points: { show: false },
        paths: (u: uPlot, seriesIdx: number, idx0: number, idx1: number) => {
          const spline = uPlot.paths.spline!()
          return spline(u, seriesIdx, idx0, idx1)
        },
        value: (_self: uPlot, val: number | null) =>
          val != null ? `${val.toFixed(2)} ${s.unit}` : '--',
      })),
    ],
    scales,
    legend: {
      show: true,
    },
  }
}

export function ProbeChart({ series, height = 400 }: ProbeChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<uPlot | null>(null)
  const [containerWidth, setContainerWidth] = useState(0)
  const prevSeriesCountRef = useRef(0)

  // Track container width
  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width)
      }
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  // Create/recreate chart when series count or container width changes
  useEffect(() => {
    if (!containerRef.current || containerWidth === 0 || series.length === 0) return

    const data = buildUPlotData(series)
    if (data[0].length === 0) return

    const seriesCountChanged = series.length !== prevSeriesCountRef.current
    prevSeriesCountRef.current = series.length

    if (chartRef.current && !seriesCountChanged) {
      // Just update data and resize
      chartRef.current.setSize({ width: containerWidth, height })
      chartRef.current.setData(data)
      return
    }

    // Destroy old chart
    if (chartRef.current) {
      chartRef.current.destroy()
      chartRef.current = null
    }

    const opts = buildOpts(series, containerWidth, height)
    chartRef.current = new uPlot(opts, data, containerRef.current)

    return () => {
      if (chartRef.current) {
        chartRef.current.destroy()
        chartRef.current = null
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [series, containerWidth, height])

  const isLoading = series.length === 0

  return (
    <div className="bg-surface-container rounded-2xl p-5 relative">
      {isLoading && (
        <div
          className="absolute inset-0 flex items-center justify-center z-10 bg-surface-container rounded-2xl"
        >
          <div className="flex flex-col items-center gap-3">
            <div className="h-8 w-8 rounded-full bg-primary/20 animate-bio-pulse" />
            <span className="text-xs text-on-surface-faint uppercase tracking-widest">
              Loading chart data...
            </span>
          </div>
        </div>
      )}
      <div
        ref={containerRef}
        style={{ minHeight: height }}
        className={cn(
          '[&_.u-legend]:!bg-transparent [&_.u-legend]:!border-0 [&_.u-legend]:!font-[Manrope,system-ui,sans-serif] [&_.u-legend]:!text-xs',
          '[&_.u-legend_.u-series]:!px-2 [&_.u-legend_.u-series]:!py-0.5',
          '[&_.u-legend_.u-label]:!text-on-surface-dim',
          '[&_.u-legend_.u-value]:!text-on-surface',
          '[&_.u-select]:!bg-primary/10',
        )}
      />
    </div>
  )
}

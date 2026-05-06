import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Check } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import { useMeasurementParameters } from '@/hooks/useMeasurements'
import { cn } from '@/lib/utils'

const SERIES_COLORS = ['#3adffa', '#6dfe9c', '#ff8796', '#c4b5fd']

interface ProbeSelectorProps {
  selected: string[]
  onChange: (probes: string[]) => void
  maxSelections?: number
}

export function ProbeSelector({
  selected,
  onChange,
  maxSelections = 4,
}: ProbeSelectorProps) {
  const { data } = useProbes()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const probes = data?.probes ?? []

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const firstProbe = selected.length === 1 ? probes.find((p) => p.name === selected[0]) : null

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 bg-surface-container-high rounded-xl px-3 py-2 text-sm transition-fluid hover:bg-surface-container-highest"
      >
        {selected.length > 0 && (
          <span className="flex items-center gap-0.5">
            {selected.slice(0, 4).map((name, i) => (
              <span
                key={name}
                className="h-2 w-2 rounded-full flex-shrink-0"
                style={{ backgroundColor: SERIES_COLORS[i % SERIES_COLORS.length] }}
              />
            ))}
          </span>
        )}
        <span className={selected.length > 0 ? 'text-on-surface' : 'text-on-surface-dim'}>
          {selected.length === 0
            ? 'Probes'
            : selected.length === 1
              ? (firstProbe?.display_name ?? selected[0])
              : `Probes (${selected.length})`}
        </span>
        <ChevronDown
          className={cn('h-3.5 w-3.5 text-on-surface-faint transition-fluid', open && 'rotate-180')}
        />
      </button>

      {open && (
        <div className="absolute z-20 top-full mt-1.5 left-0 min-w-[200px] bg-surface-container-high rounded-2xl shadow-abyss overflow-hidden">
          <div className="max-h-72 overflow-y-auto p-1.5">
          {selected.length > 0 && (
            <button
              onClick={() => onChange([])}
              className="w-full text-left px-3 py-1.5 text-xs text-on-surface-faint rounded-xl hover:bg-surface-container-highest transition-fluid mb-0.5"
            >
              Clear all
            </button>
          )}
          {probes.length === 0 ? (
            <p className="text-xs text-on-surface-faint px-3 py-2">No probes available</p>
          ) : (
            probes.map((p) => {
              const isSelected = selected.includes(p.name)
              const colorIdx = selected.indexOf(p.name)
              return (
                <button
                  key={p.name}
                  disabled={!isSelected && selected.length >= maxSelections}
                  onClick={() => {
                    if (isSelected) {
                      onChange(selected.filter((s) => s !== p.name))
                    } else {
                      onChange([...selected, p.name])
                      setOpen(false)
                    }
                  }}
                  className="w-full flex items-center gap-2 text-left px-3 py-2 text-sm text-on-surface rounded-xl transition-fluid hover:bg-surface-container-highest disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <span
                    className="h-2 w-2 rounded-full flex-shrink-0"
                    style={{
                      backgroundColor: isSelected
                        ? SERIES_COLORS[colorIdx % SERIES_COLORS.length]
                        : 'transparent',
                    }}
                  />
                  <span className="flex-1">
                    {p.display_name}
                    {p.unit && <span className="text-on-surface-faint ml-1">({p.unit})</span>}
                  </span>
                  {isSelected && <Check className="h-3.5 w-3.5 text-primary flex-shrink-0" />}
                </button>
              )
            })
          )}
          </div>
        </div>
      )}
    </div>
  )
}

interface MeasurementSelectorProps {
  selected: string[]
  onChange: (params: string[]) => void
  colorOffset?: number
  maxSelections?: number
}

export function MeasurementSelector({
  selected,
  onChange,
  colorOffset = 0,
  maxSelections = 4,
}: MeasurementSelectorProps) {
  const { data } = useMeasurementParameters()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const parameters = data?.parameters ?? []

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const firstParam = selected.length === 1 ? parameters.find((p) => p.name === selected[0]) : null

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 bg-surface-container-high rounded-xl px-3 py-2 text-sm transition-fluid hover:bg-surface-container-highest"
      >
        {selected.length > 0 && (
          <span className="flex items-center gap-0.5">
            {selected.slice(0, 4).map((name, i) => (
              <span
                key={name}
                className="h-2 w-2 rounded-full flex-shrink-0 border border-current/30"
                style={{
                  backgroundColor: SERIES_COLORS[(colorOffset + i) % SERIES_COLORS.length],
                }}
              />
            ))}
          </span>
        )}
        <span className={selected.length > 0 ? 'text-on-surface' : 'text-on-surface-dim'}>
          {selected.length === 0
            ? 'Tests'
            : selected.length === 1
              ? (firstParam?.name ?? selected[0])
              : `Tests (${selected.length})`}
        </span>
        <ChevronDown
          className={cn('h-3.5 w-3.5 text-on-surface-faint transition-fluid', open && 'rotate-180')}
        />
      </button>

      {open && (
        <div className="absolute z-20 top-full mt-1.5 left-0 min-w-[200px] bg-surface-container-high rounded-2xl shadow-abyss overflow-hidden">
          <div className="max-h-72 overflow-y-auto p-1.5">
          {selected.length > 0 && (
            <button
              onClick={() => onChange([])}
              className="w-full text-left px-3 py-1.5 text-xs text-on-surface-faint rounded-xl hover:bg-surface-container-highest transition-fluid mb-0.5"
            >
              Clear all
            </button>
          )}
          {parameters.length === 0 ? (
            <p className="text-xs text-on-surface-faint px-3 py-2">No parameters available</p>
          ) : (
            parameters.map((p) => {
              const isSelected = selected.includes(p.name)
              const colorIdx = selected.indexOf(p.name)
              return (
                <button
                  key={p.name}
                  disabled={!isSelected && selected.length >= maxSelections}
                  onClick={() => {
                    if (isSelected) {
                      onChange(selected.filter((s) => s !== p.name))
                    } else {
                      onChange([...selected, p.name])
                      setOpen(false)
                    }
                  }}
                  className="w-full flex items-center gap-2 text-left px-3 py-2 text-sm text-on-surface rounded-xl transition-fluid hover:bg-surface-container-highest disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <span
                    className="h-2 w-2 rounded-full flex-shrink-0 border border-current/30"
                    style={{
                      backgroundColor: isSelected
                        ? SERIES_COLORS[(colorOffset + colorIdx) % SERIES_COLORS.length]
                        : 'transparent',
                    }}
                  />
                  <span className="flex-1">
                    {p.name}
                    {p.canonical_unit && (
                      <span className="text-on-surface-faint ml-1">({p.canonical_unit})</span>
                    )}
                  </span>
                  {isSelected && <Check className="h-3.5 w-3.5 text-primary flex-shrink-0" />}
                </button>
              )
            })
          )}
          </div>
        </div>
      )}
    </div>
  )
}

export { SERIES_COLORS }

import { useState, useRef, useEffect } from 'react'
import { ChevronDown, X } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import { cn } from '@/lib/utils'

// Colors assigned to selected probes in order
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
  const available = probes.filter((p) => !selected.includes(p.name))

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 bg-surface-container-high rounded-xl px-3 py-2 text-sm text-on-surface transition-fluid hover:bg-surface-container-highest"
      >
        <span className="text-on-surface-dim">Probes</span>
        <ChevronDown
          className={cn(
            'h-3.5 w-3.5 text-on-surface-faint transition-fluid',
            open && 'rotate-180',
          )}
        />
      </button>

      {open && (
        <div className="absolute z-20 top-full mt-1.5 left-0 min-w-[200px] bg-surface-container-high rounded-2xl p-1.5 shadow-abyss">
          {available.length === 0 ? (
            <p className="text-xs text-on-surface-faint px-3 py-2">
              {selected.length >= maxSelections
                ? `Max ${maxSelections} probes`
                : 'No more probes'}
            </p>
          ) : (
            available.map((p) => (
              <button
                key={p.name}
                disabled={selected.length >= maxSelections}
                onClick={() => {
                  onChange([...selected, p.name])
                  setOpen(false)
                }}
                className="w-full text-left px-3 py-2 text-sm text-on-surface rounded-xl transition-fluid hover:bg-surface-container-highest disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {p.display_name}
                {p.unit && (
                  <span className="text-on-surface-faint ml-1">({p.unit})</span>
                )}
              </button>
            ))
          )}
        </div>
      )}

      {selected.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {selected.map((name, i) => {
            const probe = probes.find((p) => p.name === name)
            const color = SERIES_COLORS[i % SERIES_COLORS.length]
            return (
              <span
                key={name}
                className="inline-flex items-center gap-1.5 bg-surface-container-high rounded-full px-2.5 py-1 text-xs text-on-surface"
              >
                <span
                  className="h-2 w-2 rounded-full flex-shrink-0"
                  style={{ backgroundColor: color }}
                />
                {probe?.display_name ?? name}
                {probe?.unit && (
                  <span className="text-on-surface-faint">({probe.unit})</span>
                )}
                <button
                  onClick={() => onChange(selected.filter((s) => s !== name))}
                  className="text-on-surface-faint hover:text-on-surface transition-fluid"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            )
          })}
        </div>
      )}
    </div>
  )
}

export { SERIES_COLORS }

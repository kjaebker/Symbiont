import { useState, useRef, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface Option {
  value: string
  label: string
}

interface FilterDropdownProps {
  label: string
  value: string
  options: Option[]
  onChange: (value: string) => void
}

export function FilterDropdown({ label, value, options, onChange }: FilterDropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const selected = options.find((o) => o.value === value)
  const isActive = value !== ''

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
        className={cn(
          'flex items-center gap-2 rounded-xl px-3 py-2 text-sm transition-fluid',
          isActive
            ? 'bg-primary/20 text-primary'
            : 'bg-surface-container-high text-on-surface hover:bg-surface-container-highest',
        )}
      >
        <span>{selected?.label ?? label}</span>
        <ChevronDown
          className={cn(
            'h-3.5 w-3.5 transition-fluid',
            open && 'rotate-180',
            isActive ? 'text-primary/70' : 'text-on-surface-faint',
          )}
        />
      </button>

      {open && (
        <div className="absolute z-20 top-full mt-1.5 left-0 min-w-[160px] bg-surface-container-high rounded-2xl shadow-abyss overflow-hidden">
          <div className="max-h-64 overflow-y-auto p-1.5">
          {options.map((opt) => (
            <button
              key={opt.value}
              onClick={() => {
                onChange(opt.value)
                setOpen(false)
              }}
              className={cn(
                'w-full text-left px-3 py-2 text-sm rounded-xl transition-fluid',
                value === opt.value
                  ? 'text-primary bg-primary/10'
                  : 'text-on-surface hover:bg-surface-container-highest',
              )}
            >
              {opt.label}
            </button>
          ))}
          </div>
        </div>
      )}
    </div>
  )
}

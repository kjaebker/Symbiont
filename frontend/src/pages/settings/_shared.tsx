import { useState, useRef, useEffect } from 'react'
import {
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DraggableSyntheticListeners,
} from '@dnd-kit/core'
import { GripVertical, ToggleLeft, AlertTriangle, Activity, Droplets } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { InputCategory } from '@/api/types'

export type Tab = 'dashboard' | 'devices' | 'probes' | 'outlets' | 'tokens' | 'notifications' | 'backup' | 'log' | 'system' | 'tank' | 'agent'

export const tabs: { key: Tab; label: string }[] = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'devices', label: 'Devices' },
  { key: 'tank', label: 'Tank' },
  { key: 'probes', label: 'Probes' },
  { key: 'outlets', label: 'Outlets' },
  { key: 'agent', label: 'Agent' },
  { key: 'tokens', label: 'Tokens' },
  { key: 'notifications', label: 'Notifications' },
  { key: 'backup', label: 'Backup' },
  { key: 'log', label: 'Log' },
  { key: 'system', label: 'System' },
]

export const unitOptions = [
  { value: '', label: 'None' },
  { value: '°F', label: '°F (Fahrenheit)' },
  { value: '°C', label: '°C (Celsius)' },
  { value: 'pH', label: 'pH' },
  { value: 'Amps', label: 'Amps' },
  { value: 'Watts', label: 'Watts' },
  { value: 'Volts', label: 'Volts' },
  { value: 'PPM', label: 'PPM' },
  { value: 'PSU', label: 'PSU (Salinity)' },
  { value: 'mV', label: 'mV (Millivolts)' },
  { value: '%', label: '% (Percent)' },
]

// --- Shared drag sensors ---

export function useDragSensors() {
  return useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 20 } }),
  )
}

// --- Drag handle ---

export function DragHandle({ listeners, attributes }: { listeners?: DraggableSyntheticListeners; attributes?: React.HTMLAttributes<HTMLButtonElement> }) {
  return (
    <button
      className="touch-none p-1 rounded-lg text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high transition-fluid cursor-grab active:cursor-grabbing"
      {...attributes}
      {...listeners}
    >
      <GripVertical size={16} />
    </button>
  )
}

// --- Inline editable cell ---

export function EditableCell({
  value,
  onSave,
  type = 'text',
  className,
}: {
  value: string | number | null
  onSave: (val: string) => void
  type?: 'text' | 'number'
  className?: string
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(String(value ?? ''))
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editing) inputRef.current?.focus()
  }, [editing])

  function commit() {
    setEditing(false)
    if (draft !== String(value ?? '')) {
      onSave(draft)
    }
  }

  if (!editing) {
    return (
      <button
        onClick={() => {
          setDraft(String(value ?? ''))
          setEditing(true)
        }}
        className={cn(
          'text-left w-full px-2 py-1 rounded-lg hover:bg-surface-container-high transition-fluid cursor-pointer',
          className,
        )}
      >
        {value != null && value !== '' ? String(value) : <span className="text-on-surface-faint">—</span>}
      </button>
    )
  }

  return (
    <input
      ref={inputRef}
      type={type}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') commit()
        if (e.key === 'Escape') setEditing(false)
      }}
      className="w-full bg-surface-container-high text-on-surface text-base rounded-lg px-2 py-1 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
    />
  )
}

// --- Unit select dropdown ---

const legacyUnitMap: Record<string, string> = {
  F: '°F',
  A: 'Amps',
  W: 'Watts',
  V: 'Volts',
}

export function normalizeUnit(raw: string): string {
  return legacyUnitMap[raw] ?? raw
}

export function UnitSelect({
  value,
  onSave,
}: {
  value: string
  onSave: (val: string) => void
}) {
  const normalized = normalizeUnit(value)
  const knownValues = new Set(unitOptions.map((o) => o.value))

  return (
    <select
      value={normalized}
      onChange={(e) => onSave(e.target.value)}
      className="w-full bg-transparent text-on-surface text-base rounded-lg px-2 py-1 outline-none hover:bg-surface-container-high focus:ring-1 focus:ring-primary/30 transition-fluid cursor-pointer appearance-none"
    >
      {!knownValues.has(normalized) && normalized !== '' && (
        <option value={normalized} className="bg-surface-container text-on-surface">
          {normalized}
        </option>
      )}
      {unitOptions.map((opt) => (
        <option key={opt.value} value={opt.value} className="bg-surface-container text-on-surface">
          {opt.label}
        </option>
      ))}
    </select>
  )
}

// --- Binary input category select ---

export const binaryCategoryOptions: { value: InputCategory; label: string }[] = [
  { value: 'switch', label: 'Switch' },
  { value: 'fluid', label: 'Fluid Sensor' },
  { value: 'alarm', label: 'Alarm' },
  { value: 'virtual', label: 'Virtual' },
]

export function getCategoryIcon(category: InputCategory): { Icon: typeof ToggleLeft; color: string; bg: string } {
  switch (category) {
    case 'fluid': return { Icon: Droplets, color: 'text-amber-400', bg: 'bg-amber-400/10' }
    case 'alarm': return { Icon: AlertTriangle, color: 'text-tertiary', bg: 'bg-tertiary/10' }
    case 'virtual': return { Icon: Activity, color: 'text-primary', bg: 'bg-primary/10' }
    default: return { Icon: ToggleLeft, color: 'text-on-surface-dim', bg: 'bg-surface-container-high' }
  }
}

export function CategorySelect({ value, onSave }: { value: InputCategory; onSave: (val: string) => void }) {
  return (
    <select
      value={value}
      onChange={(e) => onSave(e.target.value)}
      className="w-full bg-transparent text-on-surface text-sm rounded-lg px-2 py-1 outline-none hover:bg-surface-container-high focus:ring-1 focus:ring-primary/30 transition-fluid cursor-pointer appearance-none"
    >
      {binaryCategoryOptions.map((opt) => (
        <option key={opt.value} value={opt.value} className="bg-surface-container text-on-surface">
          {opt.label}
        </option>
      ))}
    </select>
  )
}

// --- Loading / Empty states ---

export function LoadingState({ label }: { label: string }) {
  return (
    <div className="p-12 flex items-center justify-center">
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 rounded-full bg-primary/20 animate-bio-pulse" />
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">{label}</span>
      </div>
    </div>
  )
}

export function EmptyState({ icon, message }: { icon: React.ReactNode; message: string }) {
  return (
    <div className="p-12 flex flex-col items-center justify-center gap-3">
      <div className="text-on-surface-faint">{icon}</div>
      <span className="text-on-surface-dim text-sm text-center max-w-sm">{message}</span>
    </div>
  )
}

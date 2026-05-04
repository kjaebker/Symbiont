import { useState, useEffect } from 'react'
import { Plus, Trash2, FlaskConical } from 'lucide-react'
import {
  useMeasurementParameters,
  useMeasurements,
  useCreateMeasurement,
  useDeleteMeasurement,
  useKitCatalog,
} from '@/hooks/useMeasurements'
import { getKitPref, setKitPref } from '@/api/client'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import type { MeasurementParameter, KitDef } from '@/api/types'

// Convert a local datetime-local input value to an ISO 8601 UTC string.
function localInputToISO(value: string): string {
  if (!value) return new Date().toISOString()
  return new Date(value).toISOString()
}

// Format an ISO string to a readable local date+time.
function formatMeasuredAt(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) +
    ' ' +
    d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

// Return the current datetime in the format required by datetime-local inputs.
function nowForInput(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

interface AddFormProps {
  parameters: MeasurementParameter[]
  onClose: () => void
}

const PARAM_PLACEHOLDERS: Record<string, string> = {
  Alkalinity: '8.3',
  Calcium: '420',
  Magnesium: '1350',
  Nitrate: '2',
  Nitrite: '0',
  Phosphate: '0.05',
  Salinity: '35.0',
  pH: '8.2',
  Ammonia: '0',
  Silicate: '0.5',
}

function AddForm({ parameters, onClose }: AddFormProps) {
  const create = useCreateMeasurement()
  const { data: kitData } = useKitCatalog()
  const kitsByParam = kitData?.kits ?? {}

  const [paramName, setParamName] = useState(parameters[0]?.name ?? '')
  const [kitRef, setKitRef] = useState<string>(() => getKitPref(parameters[0]?.name ?? '') ?? '')
  const [rawValue, setRawValue] = useState('')
  const [value, setValue] = useState('')
  const [valueOverridden, setValueOverridden] = useState(false)
  const [measuredAt, setMeasuredAt] = useState(nowForInput())
  const [notes, setNotes] = useState('')
  const [error, setError] = useState('')

  const selectedParam = parameters.find((p) => p.name === paramName)
  const availableKits: KitDef[] = kitsByParam[paramName] ?? []
  const selectedKit = availableKits.find((k) => k.ref === kitRef) ?? null

  // When the parameter changes, restore kit preference.
  function handleParamChange(name: string) {
    setParamName(name)
    const pref = getKitPref(name)
    const kitsForParam = kitsByParam[name] ?? []
    const validPref = pref && kitsForParam.some((k) => k.ref === pref) ? pref : ''
    setKitRef(validPref)
    setRawValue('')
    setValue('')
    setValueOverridden(false)
  }

  // When kit changes, reset raw/value fields and persist preference.
  function handleKitChange(ref: string) {
    setKitRef(ref)
    setRawValue('')
    setValue('')
    setValueOverridden(false)
    if (ref) setKitPref(paramName, ref)
  }

  // Recompute canonical value when raw changes (unless user overrode it).
  useEffect(() => {
    if (!selectedKit || !rawValue || valueOverridden) return
    const raw = parseFloat(rawValue)
    if (isNaN(raw)) { setValue(''); return }
    const canonical = raw * selectedKit.scale + selectedKit.offset
    setValue(canonical.toFixed(4).replace(/\.?0+$/, ''))
  }, [rawValue, selectedKit, valueOverridden])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    const numVal = parseFloat(value)
    if (isNaN(numVal)) {
      setError('Value must be a number.')
      return
    }

    const rawNum = selectedKit && rawValue ? parseFloat(rawValue) : undefined

    try {
      await create.mutateAsync({
        parameter: paramName,
        value: numVal,
        measured_at: localInputToISO(measuredAt),
        notes: notes.trim() || null,
        test_kit_ref: selectedKit?.ref ?? null,
        raw_value: rawNum != null && !isNaN(rawNum) ? rawNum : null,
      })
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save measurement.')
    }
  }

  const inputCls = 'w-full bg-surface-container-high text-on-surface rounded-xl px-3 py-2.5 text-base outline-none focus:ring-1 focus:ring-primary/50 placeholder:text-on-surface-faint'
  const labelCls = 'text-xs text-on-surface-dim uppercase tracking-wider'

  return (
    <div className="bg-surface-container rounded-2xl p-6 space-y-5">
      <h2 className="text-sm font-semibold text-on-surface uppercase tracking-widest">
        New Measurement
      </h2>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Parameter */}
        <div className="space-y-1.5">
          <label className={labelCls}>Parameter</label>
          <select
            value={paramName}
            onChange={(e) => handleParamChange(e.target.value)}
            className={inputCls}
          >
            {parameters.map((p) => (
              <option key={p.id} value={p.name}>
                {p.name}{p.canonical_unit ? ` (${p.canonical_unit})` : ''}
              </option>
            ))}
          </select>
        </div>

        {/* Kit selector — only when kits exist for this parameter */}
        {availableKits.length > 0 && (
          <div className="space-y-1.5">
            <label className={labelCls}>Test Kit</label>
            <select
              value={kitRef}
              onChange={(e) => handleKitChange(e.target.value)}
              className={inputCls}
            >
              <option value="">None — enter value directly</option>
              {availableKits.map((k) => (
                <option key={k.ref} value={k.ref}>{k.name}</option>
              ))}
            </select>
          </div>
        )}

        {/* Raw value input — only when a kit is selected */}
        {selectedKit && (
          <div className="space-y-1.5">
            <label className={labelCls}>
              {selectedKit.input_label || 'Reading'}{selectedKit.input_unit ? ` (${selectedKit.input_unit})` : ''}
            </label>
            <input
              type="number"
              step="any"
              value={rawValue}
              onChange={(e) => { setRawValue(e.target.value); setValueOverridden(false) }}
              placeholder="Kit reading"
              className={inputCls}
            />
          </div>
        )}

        {/* Canonical value */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label className={labelCls}>
              {selectedKit
                ? `Converted value (${selectedParam?.canonical_unit ?? ''})`
                : `Value${selectedParam?.canonical_unit ? ` (${selectedParam.canonical_unit})` : ''}`}
            </label>
            {selectedKit && valueOverridden && (
              <span className="text-xs text-amber-400">overridden</span>
            )}
          </div>
          <input
            type="number"
            step="any"
            value={value}
            onChange={(e) => { setValue(e.target.value); setValueOverridden(true) }}
            placeholder={`e.g. ${PARAM_PLACEHOLDERS[paramName] ?? '0'}`}
            required
            className={cn(inputCls, selectedKit && !valueOverridden && value ? 'text-primary' : '')}
          />
          {selectedKit && value && !valueOverridden && (
            <p className="text-xs text-on-surface-faint">
              Auto-converted from {rawValue} {selectedKit.input_unit} → {value} {selectedParam?.canonical_unit}
            </p>
          )}
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Measured At */}
          <div className="space-y-1.5">
            <label className={labelCls}>Tested At</label>
            <input
              type="datetime-local"
              value={measuredAt}
              onChange={(e) => setMeasuredAt(e.target.value)}
              className={inputCls}
            />
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <label className={labelCls}>Notes (optional)</label>
            <input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="e.g. after 10% water change"
              className={inputCls}
            />
          </div>
        </div>

        {error && (
          <p className="text-xs text-tertiary">{error}</p>
        )}

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={create.isPending}
            className="px-5 py-2 rounded-xl bg-primary/20 text-primary text-sm font-medium hover:bg-primary/30 transition-fluid disabled:opacity-50"
          >
            {create.isPending ? 'Saving…' : 'Save Measurement'}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-5 py-2 rounded-xl text-on-surface-dim text-sm hover:text-on-surface transition-fluid"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}

interface MeasurementsProps {
  hideHeader?: boolean
  // Controlled props used when embedded (hideHeader=true)
  selectedParam?: string | null
  onSelectedParamChange?: (p: string | null) => void
  showForm?: boolean
  onShowFormChange?: (v: boolean) => void
}

export default function Measurements({
  hideHeader = false,
  selectedParam: selectedParamProp,
  onSelectedParamChange,
  showForm: showFormProp,
  onShowFormChange,
}: MeasurementsProps) {
  usePageTitle('Measurements')

  const { data: paramsData } = useMeasurementParameters()
  const parameters = paramsData?.parameters ?? []

  const [selectedParamInternal, setSelectedParamInternal] = useState<string | null>(null)
  const [showFormInternal, setShowFormInternal] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)

  const controlled = hideHeader && onSelectedParamChange !== undefined
  const selectedParam = controlled ? (selectedParamProp ?? null) : selectedParamInternal
  const setSelectedParam = controlled ? onSelectedParamChange! : setSelectedParamInternal
  const showForm = (hideHeader && onShowFormChange !== undefined) ? (showFormProp ?? false) : showFormInternal
  const setShowForm = (hideHeader && onShowFormChange !== undefined) ? onShowFormChange! : setShowFormInternal

  const deleteMutation = useDeleteMeasurement()

  const { data, isLoading } = useMeasurements(
    selectedParam ? { parameter: selectedParam } : undefined,
  )
  const measurements = data?.measurements ?? []

  async function handleDelete(id: number) {
    if (deletingId === id) {
      await deleteMutation.mutateAsync(id)
      setDeletingId(null)
    } else {
      setDeletingId(id)
    }
  }

  return (
    <div className="p-6 md:p-8 max-w-6xl mx-auto space-y-6">
      {/* Standalone header (not used when embedded in Chemistry) */}
      {!hideHeader && (
        <>
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="text-xs text-primary uppercase tracking-widest mb-2">
                Water Chemistry
              </p>
              <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
                Measurements
              </h1>
            </div>
            <button
              onClick={() => setShowForm(!showForm)}
              className={cn(
                'flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-fluid',
                showForm
                  ? 'bg-surface-container-high text-on-surface-dim'
                  : 'bg-primary/20 text-primary hover:bg-primary/30',
              )}
            >
              <Plus size={16} />
              Add Measurement
            </button>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => setSelectedParam(null)}
              className={cn(
                'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
                selectedParam === null
                  ? 'bg-primary/20 text-primary'
                  : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high',
              )}
            >
              All
            </button>
            {parameters.map((p) => (
              <button
                key={p.id}
                onClick={() => setSelectedParam(selectedParam === p.name ? null : p.name)}
                className={cn(
                  'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
                  selectedParam === p.name
                    ? 'bg-primary/20 text-primary'
                    : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high',
                )}
              >
                {p.name}
              </button>
            ))}
          </div>
        </>
      )}

      {/* Add form */}
      {showForm && parameters.length > 0 && (
        <AddForm parameters={parameters} onClose={() => setShowForm(false)} />
      )}

      {/* Table */}
      {isLoading ? (
        <div className="bg-surface-container rounded-2xl p-12 flex items-center justify-center">
          <span className="text-on-surface-faint text-sm">Loading…</span>
        </div>
      ) : measurements.length === 0 ? (
        <div className="bg-surface-container rounded-2xl p-12 flex flex-col items-center justify-center gap-3">
          <FlaskConical size={28} className="text-on-surface-faint" />
          <span className="text-on-surface-faint text-sm">
            {selectedParam
              ? `No ${selectedParam} measurements yet.`
              : 'No measurements recorded yet.'}
          </span>
          <button
            onClick={() => setShowForm(true)}
            className="text-xs text-primary hover:underline"
          >
            Add your first measurement
          </button>
        </div>
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {measurements.map((m) => (
              <div key={m.id} className="bg-surface-container rounded-2xl px-4 py-3 flex items-start gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-baseline gap-2 flex-wrap">
                    <span className="text-base font-bold font-mono text-on-surface tabular-nums">
                      {m.value % 1 === 0 ? m.value.toFixed(0) : m.value.toFixed(2)}
                      {m.canonical_unit && (
                        <span className="text-xs text-on-surface-faint font-normal ml-1">
                          {m.canonical_unit}
                        </span>
                      )}
                    </span>
                    <span className="text-sm font-medium text-on-surface">{m.parameter}</span>
                  </div>
                  <div className="text-xs text-on-surface-dim mt-0.5">{formatMeasuredAt(m.measured_at)}</div>
                  {m.notes && <div className="text-xs text-on-surface-faint mt-0.5">{m.notes}</div>}
                </div>
                <button
                  onClick={() => handleDelete(m.id)}
                  disabled={deleteMutation.isPending && deletingId === m.id}
                  className={cn(
                    'p-1.5 rounded-lg transition-fluid text-xs shrink-0',
                    deletingId === m.id
                      ? 'bg-tertiary/20 text-tertiary'
                      : 'text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10',
                  )}
                  title={deletingId === m.id ? 'Click again to confirm' : 'Delete'}
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
          {/* Desktop table */}
          <div className="hidden sm:block bg-surface-container rounded-2xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-outline-variant/20">
                <th className="text-left px-5 py-3.5 text-xs text-on-surface-dim uppercase tracking-widest font-medium">
                  Date
                </th>
                <th className="text-left px-5 py-3.5 text-xs text-on-surface-dim uppercase tracking-widest font-medium">
                  Parameter
                </th>
                <th className="text-right px-5 py-3.5 text-xs text-on-surface-dim uppercase tracking-widest font-medium">
                  Value
                </th>
                <th className="text-left px-5 py-3.5 text-xs text-on-surface-dim uppercase tracking-widest font-medium hidden md:table-cell">
                  Notes
                </th>
                <th className="px-5 py-3.5" />
              </tr>
            </thead>
            <tbody>
              {measurements.map((m, i) => (
                <tr
                  key={m.id}
                  className={cn(
                    'transition-fluid',
                    i < measurements.length - 1 && 'border-b border-outline-variant/10',
                    'hover:bg-surface-container-high/50',
                  )}
                >
                  <td className="px-5 py-3.5 text-on-surface-dim">
                    {formatMeasuredAt(m.measured_at)}
                  </td>
                  <td className="px-5 py-3.5 text-on-surface font-medium">
                    {m.parameter}
                  </td>
                  <td className="px-5 py-3.5 text-right font-mono tabular-nums">
                    <span className="text-on-surface text-base font-bold">
                      {m.value % 1 === 0 ? m.value.toFixed(0) : m.value.toFixed(2)}
                    </span>
                    {m.canonical_unit && (
                      <span className="text-xs text-on-surface-faint ml-1">
                        {m.canonical_unit}
                      </span>
                    )}
                  </td>
                  <td className="px-5 py-3.5 text-on-surface-dim text-xs hidden md:table-cell">
                    {m.notes ?? <span className="text-on-surface-faint">—</span>}
                  </td>
                  <td className="px-3 py-3.5 text-right">
                    <button
                      onClick={() => handleDelete(m.id)}
                      disabled={deleteMutation.isPending && deletingId === m.id}
                      className={cn(
                        'p-1.5 rounded-lg transition-fluid text-xs',
                        deletingId === m.id
                          ? 'bg-tertiary/20 text-tertiary'
                          : 'text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10',
                      )}
                      title={deletingId === m.id ? 'Click again to confirm' : 'Delete'}
                    >
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          </div>
        </>
      )}
    </div>
  )
}

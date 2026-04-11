import { useState } from 'react'
import { Plus, Trash2, FlaskConical } from 'lucide-react'
import {
  useMeasurementParameters,
  useMeasurements,
  useCreateMeasurement,
  useDeleteMeasurement,
} from '@/hooks/useMeasurements'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import type { MeasurementParameter } from '@/api/types'

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

function AddForm({ parameters, onClose }: AddFormProps) {
  const create = useCreateMeasurement()
  const [paramName, setParamName] = useState(parameters[0]?.name ?? '')
  const [value, setValue] = useState('')
  const [measuredAt, setMeasuredAt] = useState(nowForInput())
  const [notes, setNotes] = useState('')
  const [error, setError] = useState('')

  const selectedParam = parameters.find((p) => p.name === paramName)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    const numVal = parseFloat(value)
    if (isNaN(numVal)) {
      setError('Value must be a number.')
      return
    }

    try {
      await create.mutateAsync({
        parameter: paramName,
        value: numVal,
        measured_at: localInputToISO(measuredAt),
        notes: notes.trim() || null,
      })
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save measurement.')
    }
  }

  return (
    <div className="bg-surface-container rounded-2xl p-6 space-y-5">
      <h2 className="text-sm font-semibold text-on-surface uppercase tracking-widest">
        New Measurement
      </h2>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Parameter */}
          <div className="space-y-1.5">
            <label className="text-xs text-on-surface-dim uppercase tracking-wider">
              Parameter
            </label>
            <select
              value={paramName}
              onChange={(e) => setParamName(e.target.value)}
              className="w-full bg-surface-container-high text-on-surface rounded-xl px-3 py-2.5 text-sm outline-none focus:ring-1 focus:ring-primary/50"
            >
              {parameters.map((p) => (
                <option key={p.id} value={p.name}>
                  {p.name}{p.canonical_unit ? ` (${p.canonical_unit})` : ''}
                </option>
              ))}
            </select>
          </div>

          {/* Value */}
          <div className="space-y-1.5">
            <label className="text-xs text-on-surface-dim uppercase tracking-wider">
              Value{selectedParam?.canonical_unit ? ` (${selectedParam.canonical_unit})` : ''}
            </label>
            <input
              type="number"
              step="any"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="e.g. 1350"
              required
              className="w-full bg-surface-container-high text-on-surface rounded-xl px-3 py-2.5 text-sm outline-none focus:ring-1 focus:ring-primary/50 placeholder:text-on-surface-faint"
            />
          </div>

          {/* Measured At */}
          <div className="space-y-1.5">
            <label className="text-xs text-on-surface-dim uppercase tracking-wider">
              Tested At
            </label>
            <input
              type="datetime-local"
              value={measuredAt}
              onChange={(e) => setMeasuredAt(e.target.value)}
              className="w-full bg-surface-container-high text-on-surface rounded-xl px-3 py-2.5 text-sm outline-none focus:ring-1 focus:ring-primary/50"
            />
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <label className="text-xs text-on-surface-dim uppercase tracking-wider">
              Notes (optional)
            </label>
            <input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="e.g. after 10% water change"
              className="w-full bg-surface-container-high text-on-surface rounded-xl px-3 py-2.5 text-sm outline-none focus:ring-1 focus:ring-primary/50 placeholder:text-on-surface-faint"
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

export default function Measurements() {
  usePageTitle('Measurements')

  const { data: paramsData } = useMeasurementParameters()
  const parameters = paramsData?.parameters ?? []

  const [selectedParam, setSelectedParam] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)

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
      {/* Header */}
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
          onClick={() => { setShowForm((v) => !v) }}
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

      {/* Add form */}
      {showForm && parameters.length > 0 && (
        <AddForm parameters={parameters} onClose={() => setShowForm(false)} />
      )}

      {/* Parameter filter pills */}
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
        <div className="bg-surface-container rounded-2xl overflow-hidden">
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
      )}
    </div>
  )
}

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
import { Button, Card, EmptyState, Field, Input, LoadingState, PageBody, SectionHeader, Select, labelClass } from '@/components/ui'
import { ConfirmButton } from '@/components/ConfirmButton'
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

  return (
    <Card className="space-y-5">
      <SectionHeader title="New Measurement" />

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Parameter */}
        <Field label="Parameter">
          <Select
            value={paramName}
            onChange={(e) => handleParamChange(e.target.value)}
          >
            {parameters.map((p) => (
              <option key={p.id} value={p.name}>
                {p.name}{p.canonical_unit ? ` (${p.canonical_unit})` : ''}
              </option>
            ))}
          </Select>
        </Field>

        {/* Kit selector — only when kits exist for this parameter */}
        {availableKits.length > 0 && (
          <Field label="Test Kit">
            <Select
              value={kitRef}
              onChange={(e) => handleKitChange(e.target.value)}
            >
              <option value="">None — enter value directly</option>
              {availableKits.map((k) => (
                <option key={k.ref} value={k.ref}>{k.name}</option>
              ))}
            </Select>
          </Field>
        )}

        {/* Raw value input — only when a kit is selected */}
        {selectedKit && (
          <Field label={`${selectedKit.input_label || 'Reading'}${selectedKit.input_unit ? ` (${selectedKit.input_unit})` : ''}`}>
            <Input
              type="number"
              step="any"
              value={rawValue}
              onChange={(e) => { setRawValue(e.target.value); setValueOverridden(false) }}
              placeholder="Kit reading"
            />
          </Field>
        )}

        {/* Canonical value */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <span className={labelClass}>
              {selectedKit
                ? `Converted value (${selectedParam?.canonical_unit ?? ''})`
                : `Value${selectedParam?.canonical_unit ? ` (${selectedParam.canonical_unit})` : ''}`}
            </span>
            {selectedKit && valueOverridden && (
              <span className="text-xs text-warning">overridden</span>
            )}
          </div>
          <Input
            type="number"
            step="any"
            value={value}
            onChange={(e) => { setValue(e.target.value); setValueOverridden(true) }}
            placeholder={`e.g. ${PARAM_PLACEHOLDERS[paramName] ?? '0'}`}
            required
            aria-label="Value"
            className={selectedKit && !valueOverridden && value ? 'text-primary' : ''}
          />
          {selectedKit && value && !valueOverridden && (
            <p className="text-xs text-on-surface-faint mt-1">
              Auto-converted from {rawValue} {selectedKit.input_unit} → {value} {selectedParam?.canonical_unit}
            </p>
          )}
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Measured At */}
          <Field label="Tested At">
            <Input
              type="datetime-local"
              value={measuredAt}
              onChange={(e) => setMeasuredAt(e.target.value)}
            />
          </Field>

          {/* Notes */}
          <Field label="Notes (optional)">
            <Input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="e.g. after 10% water change"
            />
          </Field>
        </div>

        {error && (
          <p className="text-xs text-tertiary">{error}</p>
        )}

        <div className="flex items-center justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="primary" disabled={create.isPending}>
            {create.isPending ? 'Saving…' : 'Save Measurement'}
          </Button>
        </div>
      </form>
    </Card>
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

  return (
    <PageBody>
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
            <Button
              variant={showForm ? 'secondary' : 'tonal'}
              icon={Plus}
              onClick={() => setShowForm(!showForm)}
            >
              Add Measurement
            </Button>
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
        <Card flush>
          <LoadingState label="Loading measurements..." />
        </Card>
      ) : measurements.length === 0 ? (
        <Card flush>
          <EmptyState
            icon={<FlaskConical size={28} />}
            message={selectedParam
              ? `No ${selectedParam} measurements yet.`
              : 'No measurements recorded yet.'}
            action={
              <Button variant="tonal" size="sm" onClick={() => setShowForm(true)}>
                Add your first measurement
              </Button>
            }
          />
        </Card>
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {measurements.map((m) => (
              <Card key={m.id} className="px-4 py-3 flex items-start gap-3">
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
                <ConfirmButton
                  onConfirm={() => deleteMutation.mutate(m.id)}
                  disabled={deleteMutation.isPending}
                  title="Delete"
                  className="p-1.5 rounded-lg transition-fluid shrink-0 text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10"
                >
                  <Trash2 size={14} />
                </ConfirmButton>
              </Card>
            ))}
          </div>
          {/* Desktop table */}
          <Card flush className="hidden sm:block">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-surface-container-high/50">
                <th className={cn(labelClass, 'text-left px-5 py-3.5')}>
                  Date
                </th>
                <th className={cn(labelClass, 'text-left px-5 py-3.5')}>
                  Parameter
                </th>
                <th className={cn(labelClass, 'text-right px-5 py-3.5')}>
                  Value
                </th>
                <th className={cn(labelClass, 'text-left px-5 py-3.5 hidden md:table-cell')}>
                  Notes
                </th>
                <th className="px-5 py-3.5" />
              </tr>
            </thead>
            <tbody>
              {measurements.map((m) => (
                <tr
                  key={m.id}
                  className="transition-fluid hover:bg-surface-container-high/50"
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
                    <ConfirmButton
                      onConfirm={() => deleteMutation.mutate(m.id)}
                      disabled={deleteMutation.isPending}
                      title="Delete"
                      className="p-1.5 rounded-lg transition-fluid text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10"
                    >
                      <Trash2 size={14} />
                    </ConfirmButton>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          </Card>
        </>
      )}
    </PageBody>
  )
}

import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Plus,Trash2,Check,Cpu,Sparkles } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets } from '@/hooks/useOutlets'
import { useDevices,useCreateDevice,useUpdateDevice,useDeleteDevice,useSetDeviceProbes,useSetDeviceOutlets,useDeviceSuggestions } from '@/hooks/useDevices'
import type { Device,DeviceSuggestion,DeviceOutlet } from '@/api/types'
import { LoadingState,EmptyState } from './_shared'

// =============================================================================
// Devices Tab
// =============================================================================

const deviceTypeOptions = [
  { value: '', label: 'None' },
  { value: 'heater', label: 'Heater' },
  { value: 'pump', label: 'Pump' },
  { value: 'wavemaker', label: 'Wavemaker' },
  { value: 'light', label: 'Light' },
  { value: 'skimmer', label: 'Skimmer' },
  { value: 'reactor', label: 'Reactor' },
  { value: 'doser', label: 'Doser' },
  { value: 'ato', label: 'ATO' },
  { value: 'chiller', label: 'Chiller' },
  { value: 'fan', label: 'Fan' },
  { value: 'other', label: 'Other' },
]

function DeviceForm({
  initial,
  outlets,
  probes,
  existingDevices,
  onSave,
  onCancel,
}: {
  initial?: Device
  outlets: { id: string; name: string; display_name: string; type: string }[]
  probes: { name: string; display_name: string }[]
  existingDevices: Device[]
  onSave: (data: {
    name: string
    device_type: string | null
    description: string | null
    brand: string | null
    model: string | null
    notes: string | null
    image_path: string | null
    outlet_id: string | null
    probe_names: string[]
    outlet_ids: DeviceOutlet[]
  }) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [deviceType, setDeviceType] = useState(initial?.device_type ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [brand, setBrand] = useState(initial?.brand ?? '')
  const [model, setModel] = useState(initial?.model ?? '')
  const [notes, setNotes] = useState(initial?.notes ?? '')
  const [outletId, setOutletId] = useState(initial?.outlet_id ?? '')
  const [selectedProbes, setSelectedProbes] = useState<Set<string>>(new Set(initial?.probe_names ?? []))
  const [cycleOutlets, setCycleOutlets] = useState<DeviceOutlet[]>(
    initial?.outlet_ids ?? [],
  )

  // Outlets already linked to other devices (exclude current device's outlet).
  const linkedOutletIds = new Set(
    existingDevices
      .filter((d) => d.outlet_id && d.id !== initial?.id)
      .map((d) => d.outlet_id!),
  )

  // Probes already linked to other devices.
  const linkedProbeNames = new Set(
    existingDevices
      .filter((d) => d.id !== initial?.id)
      .flatMap((d) => d.probe_names),
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    onSave({
      name: name.trim(),
      device_type: deviceType || null,
      description: description || null,
      brand: brand || null,
      model: model || null,
      notes: notes || null,
      image_path: initial?.image_path ?? null,
      outlet_id: outletId || null,
      probe_names: Array.from(selectedProbes),
      outlet_ids: cycleOutlets,
    })
  }

  function addCycleOutlet(outletId: string) {
    if (cycleOutlets.some((o) => o.outlet_id === outletId)) return
    setCycleOutlets((prev) => [
      ...prev,
      { outlet_id: outletId, label: null, color: null, sort_order: prev.length },
    ])
  }

  function removeCycleOutlet(outletId: string) {
    setCycleOutlets((prev) => prev.filter((o) => o.outlet_id !== outletId))
  }

  function updateCycleOutletLabel(outletId: string, label: string) {
    setCycleOutlets((prev) =>
      prev.map((o) => (o.outlet_id === outletId ? { ...o, label: label || null } : o)),
    )
  }

  function updateCycleOutletColor(outletId: string, color: string | null) {
    setCycleOutlets((prev) =>
      prev.map((o) => (o.outlet_id === outletId ? { ...o, color } : o)),
    )
  }

  function toggleProbe(probeName: string) {
    setSelectedProbes((prev) => {
      const next = new Set(prev)
      if (next.has(probeName)) next.delete(probeName)
      else next.add(probeName)
      return next
    })
  }

  const inputClass = 'w-full bg-surface-container-high text-on-surface text-base rounded-lg px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid'
  const labelClass = 'text-xs text-on-surface-faint uppercase tracking-widest font-medium'

  return (
    <form onSubmit={handleSubmit} className="space-y-4 p-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className={labelClass}>Name *</label>
          <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} required />
        </div>
        <div>
          <label className={labelClass}>Type</label>
          <select value={deviceType} onChange={(e) => setDeviceType(e.target.value)} className={cn(inputClass, 'cursor-pointer appearance-none')}>
            {deviceTypeOptions.map((o) => (
              <option key={o.value} value={o.value} className="bg-surface-container text-on-surface">{o.label}</option>
            ))}
          </select>
        </div>
        <div>
          <label className={labelClass}>Brand</label>
          <input value={brand} onChange={(e) => setBrand(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Model</label>
          <input value={model} onChange={(e) => setModel(e.target.value)} className={inputClass} />
        </div>
        <div className="md:col-span-2">
          <label className={labelClass}>Description</label>
          <input value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />
        </div>
        <div className="md:col-span-2">
          <label className={labelClass}>Notes</label>
          <textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} className={cn(inputClass, 'resize-none')} />
        </div>
      </div>

      <div>
        <label className={labelClass}>Linked Outlet</label>
        <select value={outletId} onChange={(e) => setOutletId(e.target.value)} className={cn(inputClass, 'cursor-pointer appearance-none')}>
          <option value="" className="bg-surface-container text-on-surface">None</option>
          {outlets
            .filter((o) => o.type === 'outlet' || o.type === 'virtual')
            .map((o) => (
              <option
                key={o.id}
                value={o.id}
                disabled={linkedOutletIds.has(o.id)}
                className="bg-surface-container text-on-surface"
              >
                {o.display_name} ({o.id}){linkedOutletIds.has(o.id) ? ' — linked' : ''}
              </option>
            ))}
        </select>
      </div>

      <div>
        <label className={labelClass}>Visualization Channels</label>
        <p className="text-xs text-on-surface-dim mb-2">
          Variable outlets (0-10V) to overlay as a day cycle chart on the device card. Ideal for lights with separate color/power channels.
        </p>
        <div className="space-y-2 mb-2">
          {cycleOutlets.map((co) => {
            const outlet = outlets.find((o) => o.id === co.outlet_id)
            const PRESET_COLORS = ['#3adffa', '#6dfe9c', '#ff8796', '#fbbf24']
            return (
              <div key={co.outlet_id} className="flex items-center gap-2 bg-surface-container-high/50 rounded-xl px-3 py-2">
                <span className="text-sm text-on-surface truncate flex-1">
                  {outlet?.display_name ?? co.outlet_id}
                </span>
                <input
                  type="text"
                  placeholder="Label"
                  value={co.label ?? ''}
                  onChange={(e) => updateCycleOutletLabel(co.outlet_id, e.target.value)}
                  className="w-24 bg-surface-container-highest text-on-surface text-base rounded-lg px-2 py-1 outline-none focus:ring-1 focus:ring-primary/30"
                />
                <div className="flex gap-1">
                  {PRESET_COLORS.map((c) => (
                    <button
                      key={c}
                      type="button"
                      onClick={() => updateCycleOutletColor(co.outlet_id, co.color === c ? null : c)}
                      className="w-4 h-4 rounded-full transition-fluid"
                      style={{
                        background: c,
                        outline: co.color === c ? `2px solid ${c}` : '2px solid transparent',
                        outlineOffset: '1px',
                      }}
                    />
                  ))}
                </div>
                <button
                  type="button"
                  onClick={() => removeCycleOutlet(co.outlet_id)}
                  className="text-on-surface-faint hover:text-tertiary transition-fluid"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            )
          })}
        </div>
        <select
          value=""
          onChange={(e) => { if (e.target.value) addCycleOutlet(e.target.value) }}
          className={cn(inputClass, 'cursor-pointer appearance-none')}
        >
          <option value="" className="bg-surface-container text-on-surface">+ Add channel…</option>
          {outlets
            .filter((o) => !cycleOutlets.some((co) => co.outlet_id === o.id))
            .map((o) => (
              <option key={o.id} value={o.id} className="bg-surface-container text-on-surface">
                {o.display_name} ({o.id})
              </option>
            ))}
        </select>
      </div>

      <div>
        <label className={labelClass}>Linked Probes</label>
        <p className="text-xs text-on-surface-dim mb-2">
          Probes get unit suffix in display name, e.g. &quot;Heater (watts)&quot;
        </p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-1.5">
          {probes.map((p) => {
            const linked = linkedProbeNames.has(p.name)
            const checked = selectedProbes.has(p.name)
            return (
              <label
                key={p.name}
                className={cn(
                  'flex items-center gap-2 px-3 py-2 rounded-xl text-sm transition-fluid cursor-pointer',
                  linked && !checked ? 'opacity-40 cursor-not-allowed' : '',
                  checked ? 'bg-primary/10 text-primary' : 'bg-surface-container-high/50 text-on-surface-dim hover:bg-surface-container-high',
                )}
              >
                <input
                  type="checkbox"
                  checked={checked}
                  disabled={linked && !checked}
                  onChange={() => toggleProbe(p.name)}
                  className="sr-only"
                />
                <span className={cn('h-4 w-4 rounded flex items-center justify-center text-xs shrink-0', checked ? 'bg-primary text-on-primary' : 'bg-surface-container-highest')}>
                  {checked && <Check size={12} />}
                </span>
                <span className="truncate">{p.display_name}</span>
              </label>
            )
          })}
        </div>
      </div>

      <div className="flex gap-2 pt-2">
        <button
          type="submit"
          className="px-4 py-2 rounded-xl text-sm font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
        >
          {initial ? 'Save Changes' : 'Create Device'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 rounded-xl text-sm font-semibold text-on-surface-faint bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

export default function DevicesTab() {
  const { data: devicesData, isLoading: devicesLoading } = useDevices()
  const { data: outletsData } = useOutlets()
  const { data: probesData } = useProbes()
  const { data: suggestionsData, refetch: fetchSuggestions, isFetching: suggestionsLoading } = useDeviceSuggestions()
  const createMutation = useCreateDevice()
  const updateMutation = useUpdateDevice()
  const deleteMutation = useDeleteDevice()
  const setProbeMutation = useSetDeviceProbes()
  const setOutletsMutation = useSetDeviceOutlets()

  const [showForm, setShowForm] = useState(false)
  const [editingDevice, setEditingDevice] = useState<Device | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [showSuggestions, setShowSuggestions] = useState(false)

  const devices = devicesData?.devices ?? []
  const outlets = outletsData?.outlets ?? []
  const probes = probesData?.probes ?? []
  const suggestions = suggestionsData?.suggestions ?? []

  function handleCreate(data: Parameters<typeof createMutation.mutate>[0]) {
    createMutation.mutate(data as any, {
      onSuccess: () => setShowForm(false),
    })
  }

  function handleUpdate(data: Parameters<typeof createMutation.mutate>[0]) {
    if (!editingDevice) return
    updateMutation.mutate({ id: editingDevice.id, device: data as any }, {
      onSuccess: () => {
        // If probes changed, update them separately.
        const oldProbes = new Set(editingDevice.probe_names)
        const newProbes = new Set(data.probe_names)
        const probesChanged = oldProbes.size !== newProbes.size || [...oldProbes].some((p) => !newProbes.has(p))
        if (probesChanged) {
          setProbeMutation.mutate({ id: editingDevice.id, probeNames: data.probe_names })
        }
        // Always sync visualization outlets (server handles the diff).
        setOutletsMutation.mutate({ id: editingDevice.id, outlets: data.outlet_ids })
        setEditingDevice(null)
      },
    })
  }

  function handleDelete(id: number) {
    deleteMutation.mutate(id, {
      onSuccess: () => setDeleteConfirm(null),
    })
  }

  function handleAcceptSuggestion(s: DeviceSuggestion) {
    createMutation.mutate({
      name: s.suggested_name,
      device_type: null,
      description: null,
      brand: null,
      model: null,
      notes: null,
      outlet_id: s.outlet_id,
      probe_names: s.probe_names,
    } as any)
  }

  function handleFetchSuggestions() {
    setShowSuggestions(true)
    fetchSuggestions()
  }

  if (devicesLoading) return <LoadingState label="Loading devices..." />

  if (showForm) {
    return (
      <DeviceForm
        outlets={outlets}
        probes={probes}
        existingDevices={devices}
        onSave={handleCreate}
        onCancel={() => setShowForm(false)}
      />
    )
  }

  if (editingDevice) {
    return (
      <DeviceForm
        initial={editingDevice}
        outlets={outlets}
        probes={probes}
        existingDevices={devices}
        onSave={handleUpdate}
        onCancel={() => setEditingDevice(null)}
      />
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between px-4 pt-3 pb-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          Equipment ({devices.length})
        </span>
        <div className="flex gap-2">
          <button
            onClick={handleFetchSuggestions}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
          >
            <Sparkles size={14} />
            Suggest
          </button>
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
          >
            <Plus size={14} />
            Add Device
          </button>
        </div>
      </div>

      {showSuggestions && suggestions.length > 0 && (
        <div className="mx-4 mb-3 p-3 bg-secondary/5 rounded-2xl space-y-2">
          <p className="text-xs text-secondary uppercase tracking-widest font-medium">
            Suggested Devices
          </p>
          <p className="text-xs text-on-surface-dim">
            Auto-detected from outlet/probe naming patterns. Click to create.
          </p>
          <div className="space-y-1.5">
            {suggestions.map((s) => (
              <button
                key={s.outlet_id}
                onClick={() => handleAcceptSuggestion(s)}
                disabled={createMutation.isPending}
                className="w-full flex items-center justify-between px-3 py-2 rounded-xl bg-surface-container-high/50 hover:bg-surface-container-high transition-fluid cursor-pointer text-left"
              >
                <div>
                  <span className="text-sm font-medium text-on-surface">{s.suggested_name}</span>
                  <span className="ml-2 text-xs text-on-surface-faint">{s.probe_names.join(', ')}</span>
                </div>
                <Plus size={14} className="text-secondary shrink-0" />
              </button>
            ))}
          </div>
        </div>
      )}

      {showSuggestions && suggestions.length === 0 && !suggestionsLoading && (
        <div className="mx-4 mb-3 p-3 bg-surface-container-high/50 rounded-2xl">
          <p className="text-xs text-on-surface-faint">No suggestions found. All outlets may already be linked.</p>
        </div>
      )}

      {devices.length === 0 ? (
        <EmptyState
          icon={<Cpu size={32} />}
          message="No devices configured. Create one or use the Suggest button to auto-detect."
        />
      ) : (
        <div className="divide-y divide-outline-variant/15">
          {devices.map((d) => (
            <div key={d.id} className="flex items-center gap-4 px-4 py-3 transition-fluid hover:bg-surface-container-high/50">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-on-surface truncate">{d.name}</span>
                  {d.device_type && (
                    <span className="shrink-0 px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider bg-primary/10 text-primary">
                      {d.device_type}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-3 mt-0.5">
                  {(d.brand || d.model) && (
                    <span className="text-xs text-on-surface-faint truncate">
                      {[d.brand, d.model].filter(Boolean).join(' ')}
                    </span>
                  )}
                  {d.outlet_id && (
                    <span className="text-xs text-on-surface-faint font-mono">{d.outlet_id}</span>
                  )}
                  {d.probe_names.length > 0 && (
                    <span className="text-xs text-on-surface-faint">
                      {d.probe_names.length} probe{d.probe_names.length !== 1 ? 's' : ''}
                    </span>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <button
                  onClick={() => setEditingDevice(d)}
                  className="px-2.5 py-1 rounded-lg text-xs font-medium text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid cursor-pointer"
                >
                  Edit
                </button>
                {deleteConfirm === d.id ? (
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => handleDelete(d.id)}
                      className="px-2.5 py-1 rounded-lg text-xs font-medium text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
                    >
                      Confirm
                    </button>
                    <button
                      onClick={() => setDeleteConfirm(null)}
                      className="px-2.5 py-1 rounded-lg text-xs font-medium text-on-surface-faint hover:text-on-surface transition-fluid cursor-pointer"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => setDeleteConfirm(d.id)}
                    className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
                  >
                    <Trash2 size={14} />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}


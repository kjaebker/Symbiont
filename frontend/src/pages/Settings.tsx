import { useState, useRef, useEffect } from 'react'
import { usePageTitle } from '@/hooks/usePageTitle'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DraggableSyntheticListeners,
} from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
  arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Settings as SettingsIcon, Plus, Trash2, Copy, Check, Download, RefreshCw, GripVertical, Bell, Terminal, Cpu, Sparkles } from 'lucide-react'
import { cn, relativeTime, formatBytes } from '@/lib/utils'
import {
  useProbeConfigs,
  useUpdateProbeConfig,
  useOutletConfigs,
  useUpdateOutletConfig,
  useTokens,
  useCreateToken,
  useRevokeToken,
  useBackups,
  useTriggerBackup,
} from '@/hooks/useSettings'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets } from '@/hooks/useOutlets'
import { useSystemLog } from '@/hooks/useSystem'
import {
  useNotificationTargets,
  useUpsertNotificationTarget,
  useDeleteNotificationTarget,
  useTestNotifications,
} from '@/hooks/useNotifications'
import {
  useDevices,
  useCreateDevice,
  useUpdateDevice,
  useDeleteDevice,
  useSetDeviceProbes,
  useDeviceSuggestions,
} from '@/hooks/useDevices'
import type { ProbeConfig, OutletConfig, NotificationTarget, SystemLogLine, Device, DeviceSuggestion } from '@/api/types'

const unitOptions = [
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

type Tab = 'dashboard' | 'devices' | 'probes' | 'outlets' | 'tokens' | 'notifications' | 'backup' | 'log'

const tabs: { key: Tab; label: string }[] = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'devices', label: 'Devices' },
  { key: 'probes', label: 'Probes' },
  { key: 'outlets', label: 'Outlets' },
  { key: 'tokens', label: 'Tokens' },
  { key: 'notifications', label: 'Notifications' },
  { key: 'backup', label: 'Backup' },
  { key: 'log', label: 'Log' },
]

// --- Shared drag sensors ---

function useDragSensors() {
  return useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 5 } }),
  )
}

// --- Drag handle ---

function DragHandle({ listeners, attributes }: { listeners?: DraggableSyntheticListeners; attributes?: React.HTMLAttributes<HTMLButtonElement> }) {
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

function EditableCell({
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
      className="w-full bg-surface-container-high text-on-surface text-sm rounded-lg px-2 py-1 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
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

function normalizeUnit(raw: string): string {
  return legacyUnitMap[raw] ?? raw
}

function UnitSelect({
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
      className="w-full bg-transparent text-on-surface text-sm rounded-lg px-2 py-1 outline-none hover:bg-surface-container-high focus:ring-1 focus:ring-primary/30 transition-fluid cursor-pointer appearance-none"
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

// =============================================================================
// Dashboard Tab — unified drag-and-drop list driven by dashboard_items
// =============================================================================

import {
  useDashboardLayout,
  useReplaceDashboardLayout,
  useAddDashboardItem,
  useRemoveDashboardItem,
} from '@/hooks/useDashboardLayout'
import type { DashboardItem } from '@/api/types'

function SortableDashboardRow({
  item,
  displayName,
  onRemove,
  onLabelChange,
}: {
  item: DashboardItem
  displayName: string
  onRemove: (id: number) => void
  onLabelChange?: (id: number, label: string) => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: String(item.id),
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  const typeBadgeStyle =
    item.item_type === 'outlet'
      ? 'bg-primary/10 text-primary'
      : item.item_type === 'device'
        ? 'bg-tertiary/10 text-tertiary'
        : item.item_type === 'separator'
          ? 'bg-amber-400/10 text-amber-400'
          : 'bg-secondary/10 text-secondary'

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'flex items-center gap-3 py-2.5 px-4 transition-fluid hover:bg-surface-container-high/50',
        isDragging && 'opacity-50 bg-surface-container-high/30 z-10',
      )}
    >
      <DragHandle listeners={listeners} attributes={attributes} />
      <span className={cn(
        'shrink-0 px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider',
        typeBadgeStyle,
      )}>
        {item.item_type}
      </span>
      {item.item_type === 'separator' && onLabelChange ? (
        <EditableCell
          value={item.label ?? ''}
          onSave={(val) => onLabelChange(item.id, val)}
          className="flex-1 text-sm font-semibold text-on-surface"
        />
      ) : (
        <span className="flex-1 text-sm font-medium text-on-surface truncate">
          {displayName}
        </span>
      )}
      <button
        onClick={() => onRemove(item.id)}
        className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
      >
        <Trash2 size={16} />
      </button>
    </div>
  )
}

function DashboardTab() {
  const { data: layoutData, isLoading: layoutLoading } = useDashboardLayout()
  const { data: probesData, isLoading: probesLoading } = useProbes()
  const { data: outletsData, isLoading: outletsLoading } = useOutlets()
  const { data: devicesData, isLoading: devicesLoading } = useDevices()
  const replaceMutation = useReplaceDashboardLayout()
  const addMutation = useAddDashboardItem()
  const removeMutation = useRemoveDashboardItem()
  const sensors = useDragSensors()

  const [showPicker, setShowPicker] = useState(false)
  const [localItems, setLocalItems] = useState<DashboardItem[] | null>(null)

  const isLoading = layoutLoading || probesLoading || outletsLoading || devicesLoading
  const items = localItems ?? (layoutData?.items ?? [])

  // Build display name lookup.
  const probeNameMap = new Map((probesData?.probes ?? []).map((p) => [p.name, p.display_name]))
  const outletNameMap = new Map((outletsData?.outlets ?? []).map((o) => [o.id, o.display_name]))
  const deviceNameMap = new Map((devicesData?.devices ?? []).map((d) => [String(d.id), d.name]))

  function getDisplayName(item: DashboardItem): string {
    if (item.item_type === 'separator') return item.label ?? 'Section'
    const ref = item.reference_id ?? ''
    switch (item.item_type) {
      case 'probe': return probeNameMap.get(ref) ?? ref
      case 'outlet': return outletNameMap.get(ref) ?? ref
      case 'device': return deviceNameMap.get(ref) ?? ref
      default: return ref
    }
  }

  // Items already on dashboard (for picker exclusion).
  const onDashboard = new Set(
    items
      .filter((i) => i.item_type !== 'separator' && i.reference_id)
      .map((i) => `${i.item_type}:${i.reference_id}`),
  )

  // Available items for the picker.
  const availableProbes = (probesData?.probes ?? []).filter((p) => !onDashboard.has(`probe:${p.name}`))
  const availableOutlets = (outletsData?.outlets ?? [])
    .filter((o) => (o.type === 'outlet' || o.type === 'virtual') && !onDashboard.has(`outlet:${o.id}`))
  const availableDevices = (devicesData?.devices ?? []).filter((d) => !onDashboard.has(`device:${String(d.id)}`))

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) {
      setLocalItems(null)
      return
    }
    const oldIndex = items.findIndex((i) => String(i.id) === active.id)
    const newIndex = items.findIndex((i) => String(i.id) === over.id)
    const reordered = arrayMove(items, oldIndex, newIndex)
    setLocalItems(reordered)
    replaceMutation.mutate(
      reordered.map((i) => ({
        item_type: i.item_type,
        reference_id: i.reference_id,
        label: i.label,
      })),
      { onSettled: () => setTimeout(() => setLocalItems(null), 300) },
    )
  }

  function handleRemove(id: number) {
    removeMutation.mutate(id)
  }

  function handleAdd(itemType: DashboardItem['item_type'], referenceId: string) {
    addMutation.mutate({
      item_type: itemType,
      reference_id: referenceId,
      label: null,
    })
    setShowPicker(false)
  }

  function handleAddSeparator() {
    addMutation.mutate({
      item_type: 'separator',
      reference_id: null,
      label: 'New Section',
    })
  }

  function handleLabelChange(id: number, label: string) {
    // Update the label inline by replacing the full layout.
    const updated = items.map((i) => ({
      item_type: i.item_type,
      reference_id: i.reference_id,
      label: i.id === id ? label : i.label,
    }))
    replaceMutation.mutate(updated)
  }

  if (isLoading) return <LoadingState label="Loading dashboard layout..." />

  return (
    <div>
      <div className="px-4 py-3 bg-surface-container-high/30 flex items-center justify-between">
        <p className="text-xs text-on-surface-faint">
          Drag to reorder. Remove items with the trash icon.
        </p>
        <div className="flex gap-2">
          <button
            onClick={handleAddSeparator}
            className="px-3 py-1.5 text-xs font-medium rounded-full bg-amber-400/10 text-amber-400 hover:bg-amber-400/20 transition-fluid cursor-pointer"
          >
            + Separator
          </button>
          <button
            onClick={() => setShowPicker(!showPicker)}
            className="px-3 py-1.5 text-xs font-medium rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-fluid cursor-pointer"
          >
            <Plus size={14} className="inline -mt-0.5 mr-1" />
            Add Item
          </button>
        </div>
      </div>

      {/* Picker */}
      {showPicker && (
        <div className="px-4 py-3 bg-surface-container-high/20 space-y-3">
          {availableProbes.length > 0 && (
            <div>
              <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1">Probes</p>
              <div className="flex flex-wrap gap-1.5">
                {availableProbes.map((p) => (
                  <button
                    key={p.name}
                    onClick={() => handleAdd('probe', p.name)}
                    className="px-2.5 py-1 text-xs rounded-full bg-secondary/10 text-secondary hover:bg-secondary/20 transition-fluid cursor-pointer"
                  >
                    {p.display_name}
                  </button>
                ))}
              </div>
            </div>
          )}
          {availableOutlets.length > 0 && (
            <div>
              <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1">Outlets</p>
              <div className="flex flex-wrap gap-1.5">
                {availableOutlets.map((o) => (
                  <button
                    key={o.id}
                    onClick={() => handleAdd('outlet', o.id)}
                    className="px-2.5 py-1 text-xs rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-fluid cursor-pointer"
                  >
                    {o.display_name}
                  </button>
                ))}
              </div>
            </div>
          )}
          {availableDevices.length > 0 && (
            <div>
              <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1">Devices</p>
              <div className="flex flex-wrap gap-1.5">
                {availableDevices.map((d) => (
                  <button
                    key={d.id}
                    onClick={() => handleAdd('device', String(d.id))}
                    className="px-2.5 py-1 text-xs rounded-full bg-tertiary/10 text-tertiary hover:bg-tertiary/20 transition-fluid cursor-pointer"
                  >
                    {d.name}
                  </button>
                ))}
              </div>
            </div>
          )}
          {availableProbes.length === 0 && availableOutlets.length === 0 && availableDevices.length === 0 && (
            <p className="text-xs text-on-surface-faint text-center py-2">All items are already on the dashboard.</p>
          )}
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          icon={<SettingsIcon size={32} />}
          message="No items on dashboard. Use 'Add Item' to populate it."
        />
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={items.map((i) => String(i.id))} strategy={verticalListSortingStrategy}>
            {items.map((item) => (
              <SortableDashboardRow
                key={item.id}
                item={item}
                displayName={getDisplayName(item)}
                onRemove={handleRemove}
                onLabelChange={item.item_type === 'separator' ? handleLabelChange : undefined}
              />
            ))}
          </SortableContext>
        </DndContext>
      )}
    </div>
  )
}

// =============================================================================
// Probes Tab — table for configuration (no drag-and-drop)
// =============================================================================

function useMergedProbeConfigs(
  probes: { name: string; display_name: string; type: string; unit: string }[] | undefined,
  configs: ProbeConfig[] | undefined,
): ProbeConfig[] {
  if (!probes) return configs ?? []

  const configMap = new Map((configs ?? []).map((c) => [c.probe_name, c]))

  return probes.map((p) => {
    const existing = configMap.get(p.name)
    return {
      probe_name: p.name,
      display_name: existing?.display_name ?? p.display_name ?? p.name,
      unit_override: existing?.unit_override ?? p.unit ?? '',
      min_normal: existing?.min_normal ?? null,
      max_normal: existing?.max_normal ?? null,
      min_warning: existing?.min_warning ?? null,
      max_warning: existing?.max_warning ?? null,
      device_id: existing?.device_id ?? null,
    }
  })
}

function ProbesTab() {
  const { data: configData, isLoading: configsLoading } = useProbeConfigs()
  const { data: probesData, isLoading: probesLoading } = useProbes()
  const { data: devicesData } = useDevices()
  const updateMutation = useUpdateProbeConfig()

  const isLoading = configsLoading || probesLoading
  const items = useMergedProbeConfigs(probesData?.probes, configData?.configs)

  // Build device name lookup by ID for tooltip.
  const deviceNameById = new Map(
    (devicesData?.devices ?? []).map((d) => [d.id, d.name]),
  )

  function handleUpdate(name: string, field: keyof ProbeConfig, raw: string) {
    const numericFields: (keyof ProbeConfig)[] = [
      'min_normal', 'max_normal', 'min_warning', 'max_warning',
    ]
    const value = numericFields.includes(field) ? (raw === '' ? null : Number(raw)) : raw
    updateMutation.mutate({ name, config: { [field]: value } })
  }

  if (isLoading) return <LoadingState label="Loading probe configs..." />

  if (items.length === 0) {
    return (
      <EmptyState
        icon={<SettingsIcon size={32} />}
        message="No probes found. Probes will appear here after the first poll."
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="bg-surface-container-high/50">
            {['Probe', 'Display Name', 'Unit', 'Min Normal', 'Max Normal', 'Min Warn', 'Max Warn'].map((h) => (
              <th key={h} className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((c) => {
            const deviceName = c.device_id ? deviceNameById.get(c.device_id) : null
            return (
            <tr key={c.probe_name} className="transition-fluid hover:bg-surface-container-high/50">
              <td className="py-2 px-4 text-sm font-medium text-on-surface">{c.probe_name}</td>
              <td className="py-2 px-4">
                {deviceName ? (
                  <span
                    className="inline-flex items-center gap-1.5 px-2 py-1 text-sm text-on-surface-dim cursor-default"
                    title={`Managed by device "${deviceName}" — edit the device name to change this`}
                  >
                    {c.display_name}
                    <span className="text-xs text-on-surface-faint">({deviceName})</span>
                  </span>
                ) : (
                  <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                )}
              </td>
              <td className="py-2 px-4">
                <UnitSelect value={c.unit_override} onSave={(v) => handleUpdate(c.probe_name, 'unit_override', v)} />
              </td>
              <td className="py-2 px-4">
                <EditableCell value={c.min_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_normal', v)} />
              </td>
              <td className="py-2 px-4">
                <EditableCell value={c.max_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_normal', v)} />
              </td>
              <td className="py-2 px-4">
                <EditableCell value={c.min_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_warning', v)} />
              </td>
              <td className="py-2 px-4">
                <EditableCell value={c.max_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_warning', v)} />
              </td>
            </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

// =============================================================================
// Outlets Tab — table for configuration (no drag-and-drop)
// =============================================================================

function useMergedOutletConfigs(
  outlets: { id: string; name: string; display_name: string; type: string }[] | undefined,
  configs: OutletConfig[] | undefined,
): (OutletConfig & { outletName: string })[] {
  if (!outlets) return []
  const configMap = new Map((configs ?? []).map((c) => [c.outlet_id, c]))

  return outlets
    .filter((o) => o.type === 'outlet' || o.type === 'virtual')
    .map((o) => {
      const existing = configMap.get(o.id)
      return {
        outlet_id: o.id,
        display_name: existing?.display_name ?? o.display_name ?? o.name,
        icon: existing?.icon ?? '',
        outletName: o.name,
      }
    })
}

function OutletsTab() {
  const { data: outletConfigData, isLoading: outletConfigsLoading } = useOutletConfigs()
  const { data: outletsData, isLoading: outletsLoading } = useOutlets()
  const { data: devicesData } = useDevices()
  const updateOutletMutation = useUpdateOutletConfig()

  const isLoading = outletConfigsLoading || outletsLoading
  const items = useMergedOutletConfigs(outletsData?.outlets, outletConfigData?.configs)

  // Build outlet_id → device name lookup.
  const deviceNameByOutletId = new Map(
    (devicesData?.devices ?? [])
      .filter((d) => d.outlet_id)
      .map((d) => [d.outlet_id!, d.name]),
  )

  function handleUpdate(id: string, field: keyof OutletConfig, raw: string) {
    updateOutletMutation.mutate({ id, config: { [field]: raw } })
  }

  if (isLoading) return <LoadingState label="Loading outlet configs..." />

  if (items.length === 0) {
    return (
      <EmptyState
        icon={<SettingsIcon size={32} />}
        message="No outlets found. Outlets will appear here after the first poll."
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="bg-surface-container-high/50">
            {['Outlet', 'Display Name', 'ID'].map((h) => (
              <th key={h} className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const deviceName = deviceNameByOutletId.get(item.outlet_id)
            return (
              <tr key={item.outlet_id} className="transition-fluid hover:bg-surface-container-high/50">
                <td className="py-2 px-4 text-sm font-medium text-on-surface">{item.outletName}</td>
                <td className="py-2 px-4">
                  {deviceName ? (
                    <span
                      className="inline-flex items-center gap-1.5 px-2 py-1 text-sm text-on-surface-dim cursor-default"
                      title={`Managed by device "${deviceName}" — edit the device name to change this`}
                    >
                      {item.display_name}
                      <span className="text-xs text-on-surface-faint">({deviceName})</span>
                    </span>
                  ) : (
                    <EditableCell value={item.display_name} onSave={(v) => handleUpdate(item.outlet_id, 'display_name', v)} />
                  )}
                </td>
                <td className="py-2 px-4 text-xs text-on-surface-faint font-mono">{item.outlet_id}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

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
  outlets: { id: string; name: string; display_name: string }[]
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
    })
  }

  function toggleProbe(probeName: string) {
    setSelectedProbes((prev) => {
      const next = new Set(prev)
      if (next.has(probeName)) next.delete(probeName)
      else next.add(probeName)
      return next
    })
  }

  const inputClass = 'w-full bg-surface-container-high text-on-surface text-sm rounded-lg px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid'
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
          {outlets.map((o) => (
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

function DevicesTab() {
  const { data: devicesData, isLoading: devicesLoading } = useDevices()
  const { data: outletsData } = useOutlets()
  const { data: probesData } = useProbes()
  const { data: suggestionsData, refetch: fetchSuggestions, isFetching: suggestionsLoading } = useDeviceSuggestions()
  const createMutation = useCreateDevice()
  const updateMutation = useUpdateDevice()
  const deleteMutation = useDeleteDevice()
  const setProbeMutation = useSetDeviceProbes()

  const [showForm, setShowForm] = useState(false)
  const [editingDevice, setEditingDevice] = useState<Device | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [showSuggestions, setShowSuggestions] = useState(false)

  const devices = devicesData?.devices ?? []
  const outlets = (outletsData?.outlets ?? []).filter((o) => o.type === 'outlet' || o.type === 'virtual')
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
        const changed = oldProbes.size !== newProbes.size || [...oldProbes].some((p) => !newProbes.has(p))
        if (changed) {
          setProbeMutation.mutate({ id: editingDevice.id, probeNames: data.probe_names })
        }
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

// =============================================================================
// Tokens Tab
// =============================================================================

function TokensTab() {
  const { data, isLoading } = useTokens()
  const createMutation = useCreateToken()
  const revokeMutation = useRevokeToken()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [newToken, setNewToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [revokeConfirm, setRevokeConfirm] = useState<number | null>(null)

  const tokens = data?.tokens ?? []

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!label.trim()) return
    createMutation.mutate(label.trim(), {
      onSuccess: (data) => {
        setNewToken(data.token)
        setLabel('')
        setShowForm(false)
      },
    })
  }

  function handleCopy(token: string) {
    navigator.clipboard.writeText(token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  function handleRevoke(id: number) {
    revokeMutation.mutate(id, {
      onSuccess: () => setRevokeConfirm(null),
    })
  }

  return (
    <div className="space-y-4">
      {newToken && (
        <div className="bg-secondary/10 rounded-2xl p-4 space-y-2">
          <p className="text-xs text-secondary uppercase tracking-widest font-medium">
            Token Created
          </p>
          <p className="text-xs text-on-surface-dim">
            Copy this token now. It will not be shown again.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-surface-container-high rounded-lg px-3 py-2 text-sm text-on-surface font-mono break-all">
              {newToken}
            </code>
            <button
              onClick={() => handleCopy(newToken)}
              className="p-2 rounded-lg text-on-surface-faint hover:text-secondary hover:bg-secondary/10 transition-fluid cursor-pointer"
            >
              {copied ? <Check size={16} /> : <Copy size={16} />}
            </button>
          </div>
          <button
            onClick={() => setNewToken(null)}
            className="text-xs text-on-surface-faint hover:text-on-surface transition-fluid cursor-pointer"
          >
            Dismiss
          </button>
        </div>
      )}

      <div className="flex items-center justify-between px-4 pt-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          API Tokens
        </span>
        {!showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
          >
            <Plus size={14} />
            Create Token
          </button>
        )}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="flex items-center gap-2 px-4">
          <input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="Token label (e.g. CLI, MCP)"
            className="flex-1 bg-surface-container-high text-on-surface text-sm rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
            autoFocus
          />
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="px-3 py-2 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
          >
            Create
          </button>
          <button
            type="button"
            onClick={() => {
              setShowForm(false)
              setLabel('')
            }}
            className="px-3 py-2 rounded-xl text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
          >
            Cancel
          </button>
        </form>
      )}

      {isLoading ? (
        <LoadingState label="Loading tokens..." />
      ) : tokens.length === 0 ? (
        <EmptyState
          icon={<SettingsIcon size={32} />}
          message="No API tokens. Create one to authenticate CLI or MCP clients."
        />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-surface-container-high/50">
                {['ID', 'Label', 'Created', 'Last Used', ''].map((h) => (
                  <th
                    key={h}
                    className={cn(
                      'py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest',
                      h === '' ? 'text-right' : 'text-left',
                    )}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {tokens.map((t) => (
                <tr key={t.id} className="transition-fluid hover:bg-surface-container-high/50">
                  <td className="py-3 px-4 text-sm text-on-surface-dim font-mono">{t.id}</td>
                  <td className="py-3 px-4 text-sm font-medium text-on-surface">{t.label}</td>
                  <td className="py-3 px-4 text-sm text-on-surface-dim">
                    {relativeTime(t.created_at)}
                  </td>
                  <td className="py-3 px-4 text-sm text-on-surface-dim">
                    {t.last_used ? relativeTime(t.last_used) : 'Never'}
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex items-center justify-end">
                      {revokeConfirm === t.id ? (
                        <div className="flex items-center gap-1">
                          <button
                            onClick={() => handleRevoke(t.id)}
                            className="px-2 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer"
                          >
                            Revoke
                          </button>
                          <button
                            onClick={() => setRevokeConfirm(null)}
                            className="px-2 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setRevokeConfirm(t.id)}
                          className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
                        >
                          <Trash2 size={14} />
                        </button>
                      )}
                    </div>
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

// =============================================================================
// Backup Tab
// =============================================================================

function BackupTab() {
  const { data, isLoading } = useBackups()
  const triggerMutation = useTriggerBackup()

  const backups = data?.backups ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-4 pt-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          Database Backups
        </span>
        <button
          onClick={() => triggerMutation.mutate()}
          disabled={triggerMutation.isPending}
          className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
        >
          {triggerMutation.isPending ? (
            <RefreshCw size={14} className="animate-spin" />
          ) : (
            <Download size={14} />
          )}
          Run Backup Now
        </button>
      </div>

      {triggerMutation.isSuccess && (
        <div className="mx-4 bg-secondary/10 rounded-xl px-4 py-2">
          <p className="text-xs text-secondary font-medium">Backup completed successfully.</p>
        </div>
      )}

      {triggerMutation.isError && (
        <div className="mx-4 bg-tertiary/10 rounded-xl px-4 py-2">
          <p className="text-xs text-tertiary font-medium">Backup failed. Check server logs.</p>
        </div>
      )}

      {isLoading ? (
        <LoadingState label="Loading backups..." />
      ) : backups.length === 0 ? (
        <EmptyState
          icon={<Download size={32} />}
          message="No backups yet. Run a backup to create a snapshot of your databases."
        />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-surface-container-high/50">
                {['Date', 'Status', 'Size', 'Path'].map((h) => (
                  <th
                    key={h}
                    className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {backups.map((b) => (
                <tr key={b.id} className="transition-fluid hover:bg-surface-container-high/50">
                  <td className="py-3 px-4 text-sm text-on-surface-dim">
                    {relativeTime(b.ts)}
                  </td>
                  <td className="py-3 px-4">
                    <span
                      className={cn(
                        'inline-block px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider',
                        b.status === 'success'
                          ? 'bg-secondary/15 text-secondary'
                          : 'bg-tertiary/15 text-tertiary',
                      )}
                    >
                      {b.status}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-sm text-on-surface font-mono">
                    {formatBytes(b.size_bytes)}
                  </td>
                  <td className="py-3 px-4 text-sm text-on-surface-dim font-mono truncate max-w-xs">
                    {b.path}
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

// =============================================================================
// Notifications Tab
// =============================================================================

function NotificationsTab() {
  const { data, isLoading } = useNotificationTargets()
  const upsertMutation = useUpsertNotificationTarget()
  const deleteMutation = useDeleteNotificationTarget()
  const testMutation = useTestNotifications()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [url, setUrl] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  const targets = data?.targets ?? []

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!label.trim() || !url.trim()) return
    upsertMutation.mutate(
      { type: 'ntfy', label: label.trim(), config: url.trim(), enabled: true },
      {
        onSuccess: () => {
          setLabel('')
          setUrl('')
          setShowForm(false)
        },
      },
    )
  }

  function handleToggle(t: NotificationTarget) {
    upsertMutation.mutate({ id: t.id, type: t.type, label: t.label, config: t.config, enabled: !t.enabled })
  }

  function handleDelete(id: number) {
    deleteMutation.mutate(id, {
      onSuccess: () => setDeleteConfirm(null),
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-4 pt-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          ntfy.sh Targets
        </span>
        <div className="flex items-center gap-2">
          {targets.some((t) => t.enabled) && (
            <button
              onClick={() => testMutation.mutate()}
              disabled={testMutation.isPending}
              className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer disabled:opacity-50"
            >
              <Bell size={14} />
              Send Test
            </button>
          )}
          {!showForm && (
            <button
              onClick={() => setShowForm(true)}
              className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
            >
              <Plus size={14} />
              Add Target
            </button>
          )}
        </div>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="flex items-center gap-2 px-4 flex-wrap">
          <input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="Label (e.g. Phone)"
            className="flex-1 min-w-32 bg-surface-container-high text-on-surface text-sm rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
            autoFocus
          />
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://ntfy.sh/your-topic"
            className="flex-[3] min-w-48 bg-surface-container-high text-on-surface text-sm rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
          />
          <button
            type="submit"
            disabled={upsertMutation.isPending || !label.trim() || !url.trim()}
            className="px-3 py-2 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
          >
            Add
          </button>
          <button
            type="button"
            onClick={() => { setShowForm(false); setLabel(''); setUrl('') }}
            className="px-3 py-2 rounded-xl text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
          >
            Cancel
          </button>
        </form>
      )}

      {testMutation.isSuccess && (
        <div className="mx-4 space-y-1">
          {testMutation.data?.results.map((r) => (
            <div
              key={r.label}
              className={cn('rounded-xl px-4 py-2', r.success ? 'bg-secondary/10' : 'bg-tertiary/10')}
            >
              <p className={cn('text-xs font-medium', r.success ? 'text-secondary' : 'text-tertiary')}>
                {r.label}: {r.success ? 'Test notification sent.' : `Failed — ${r.error}`}
              </p>
            </div>
          ))}
        </div>
      )}

      {testMutation.isError && (
        <div className="mx-4 bg-tertiary/10 rounded-xl px-4 py-2">
          <p className="text-xs text-tertiary font-medium">
            {(testMutation.error as Error)?.message ?? 'Test failed. Check your ntfy topic URL.'}
          </p>
        </div>
      )}

      {isLoading ? (
        <LoadingState label="Loading notification targets..." />
      ) : targets.length === 0 ? (
        <EmptyState
          icon={<Bell size={32} />}
          message="No notification targets configured. Add an ntfy.sh topic URL to get alerted on your phone."
        />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-surface-container-high/50">
                {['Label', 'URL', 'Status', ''].map((h) => (
                  <th
                    key={h}
                    className={cn(
                      'py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest',
                      h === '' ? 'text-right' : 'text-left',
                    )}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {targets.map((t) => (
                <tr key={t.id} className="transition-fluid hover:bg-surface-container-high/50">
                  <td className="py-3 px-4 text-sm font-medium text-on-surface">{t.label}</td>
                  <td className="py-3 px-4 text-sm text-on-surface-dim font-mono truncate max-w-xs">{t.config}</td>
                  <td className="py-3 px-4">
                    <button
                      onClick={() => handleToggle(t)}
                      disabled={upsertMutation.isPending}
                      className={cn(
                        'px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider transition-fluid cursor-pointer disabled:opacity-50',
                        t.enabled
                          ? 'bg-secondary/15 text-secondary hover:bg-secondary/25'
                          : 'bg-surface-container-highest text-on-surface-faint hover:bg-surface-container-highest/80',
                      )}
                    >
                      {t.enabled ? 'Enabled' : 'Disabled'}
                    </button>
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex items-center justify-end">
                      {deleteConfirm === t.id ? (
                        <div className="flex items-center gap-1">
                          <button
                            onClick={() => handleDelete(t.id)}
                            className="px-2 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer"
                          >
                            Delete
                          </button>
                          <button
                            onClick={() => setDeleteConfirm(null)}
                            className="px-2 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setDeleteConfirm(t.id)}
                          className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
                        >
                          <Trash2 size={14} />
                        </button>
                      )}
                    </div>
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

// =============================================================================
// System Log Tab
// =============================================================================

const levelBadge: Record<string, string> = {
  ERROR: 'bg-tertiary/15 text-tertiary',
  WARN:  'bg-amber-400/15 text-amber-400',
  INFO:  'bg-primary/10 text-primary',
  DEBUG: 'bg-surface-container-highest text-on-surface-faint',
}

const levelText: Record<string, string> = {
  ERROR: 'text-tertiary',
  WARN:  'text-amber-400',
  INFO:  'text-on-surface-dim',
  DEBUG: 'text-on-surface-faint',
}

const serviceBadge: Record<string, string> = {
  api:    'bg-primary/10 text-primary',
  poller: 'bg-secondary/10 text-secondary',
}

function LogEntry({ line }: { line: SystemLogLine }) {
  const [expanded, setExpanded] = useState(false)
  const hasFields = line.fields && Object.keys(line.fields).length > 0

  return (
    <div
      className={cn(
        'px-4 py-1.5 hover:bg-surface-container-high/50 transition-fluid',
        hasFields && 'cursor-pointer',
      )}
      onClick={() => hasFields && setExpanded(!expanded)}
    >
      <div className="flex items-baseline gap-2 flex-wrap">
        <span className="text-on-surface-faint shrink-0 tabular-nums">
          {line.ts ? new Date(line.ts).toLocaleTimeString() : '—'}
        </span>
        <span className={cn('shrink-0 px-1.5 py-0.5 rounded text-xs font-medium uppercase', serviceBadge[line.service] ?? 'bg-surface-container-highest text-on-surface-faint')}>
          {line.service}
        </span>
        <span className={cn('shrink-0 px-1.5 py-0.5 rounded text-xs font-medium uppercase', levelBadge[line.level] ?? levelBadge.INFO)}>
          {line.level}
        </span>
        <span className={cn('flex-1 min-w-0', levelText[line.level] ?? levelText.INFO)}>
          {line.msg}
        </span>
      </div>
      {expanded && hasFields && (
        <div className="mt-1 pl-4 space-y-0.5">
          {Object.entries(line.fields!).map(([k, v]) => (
            <div key={k} className="flex gap-1">
              <span className="text-on-surface-dim">{k}</span>
              <span className="text-on-surface-faint">=</span>
              <span className="text-on-surface-dim">{String(v)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function SystemLogTab() {
  const [service, setService] = useState('')
  const { data, isLoading, refetch, isFetching } = useSystemLog(
    service ? { service, limit: 200 } : { limit: 200 },
  )

  const lines = data?.lines ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-4 pt-2">
        <div className="flex items-center gap-3">
          <span className="text-xs text-on-surface-faint uppercase tracking-widest">System Log</span>
          <div className="flex gap-1">
            {(['', 'api', 'poller'] as const).map((s) => (
              <button
                key={s}
                onClick={() => setService(s)}
                className={cn(
                  'px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider transition-fluid cursor-pointer',
                  service === s
                    ? 'bg-primary/20 text-primary'
                    : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                )}
              >
                {s || 'All'}
              </button>
            ))}
          </div>
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer disabled:opacity-50"
        >
          <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {isLoading ? (
        <LoadingState label="Loading logs..." />
      ) : lines.length === 0 ? (
        <EmptyState
          icon={<Terminal size={32} />}
          message="No log entries found. Logs are read from the systemd journal when running as a service."
        />
      ) : (
        <div className="overflow-y-auto max-h-[600px] font-mono text-xs">
          {[...lines].reverse().map((line, i) => (
            <LogEntry key={i} line={line} />
          ))}
        </div>
      )}
    </div>
  )
}

// =============================================================================
// Shared components
// =============================================================================

function LoadingState({ label }: { label: string }) {
  return (
    <div className="p-12 flex items-center justify-center">
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 rounded-full bg-primary/20 animate-bio-pulse" />
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">{label}</span>
      </div>
    </div>
  )
}

function EmptyState({ icon, message }: { icon: React.ReactNode; message: string }) {
  return (
    <div className="p-12 flex flex-col items-center justify-center gap-3">
      <div className="text-on-surface-faint">{icon}</div>
      <span className="text-on-surface-dim text-sm text-center max-w-sm">{message}</span>
    </div>
  )
}

// =============================================================================
// Settings Page
// =============================================================================

export default function Settings() {
  usePageTitle('Settings')
  const [activeTab, setActiveTab] = useState<Tab>('dashboard')

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto space-y-6">
      <div>
        <p className="text-xs text-primary uppercase tracking-widest mb-2">
          Admin Console
        </p>
        <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
          System Core Settings
        </h1>
      </div>

      <div className="flex flex-wrap gap-1.5">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={cn(
              'px-4 py-2 rounded-xl text-sm font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
              activeTab === tab.key
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container-high',
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="bg-surface-container rounded-2xl overflow-hidden">
        {activeTab === 'dashboard' && <DashboardTab />}
        {activeTab === 'devices' && <DevicesTab />}
        {activeTab === 'probes' && <ProbesTab />}
        {activeTab === 'outlets' && <OutletsTab />}
        {activeTab === 'tokens' && <TokensTab />}
        {activeTab === 'notifications' && <NotificationsTab />}
        {activeTab === 'backup' && <BackupTab />}
        {activeTab === 'log' && <SystemLogTab />}
      </div>
    </div>
  )
}

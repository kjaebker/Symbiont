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
import { Settings as SettingsIcon, Plus, Trash2, Copy, Check, Download, RefreshCw, GripVertical, Bell, Terminal, Cpu, Sparkles, LayoutGrid, LayoutList, Activity, Droplets, Waves, Bot, ChevronDown, ChevronUp, AlertTriangle, ToggleLeft, ArrowRight } from 'lucide-react'
import { cn, relativeTime, formatBytes } from '@/lib/utils'
import {
  useProbeConfigs,
  useUpdateProbeConfig,
  useOutletConfigs,
  useUpdateOutletConfig,
  useTokens,
  useCreateToken,
  useUpdateTokenScope,
  useRevokeToken,
  useBackups,
  useTriggerBackup,
} from '@/hooks/useSettings'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets } from '@/hooks/useOutlets'
import { useMeasurementParameters } from '@/hooks/useMeasurements'
import { useSystemStatus, useSystemLog } from '@/hooks/useSystem'
import { useEventBusStats } from '@/hooks/useEvents'
import { getBubblesEnabled, setBubblesEnabled } from '@/api/client'
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
import { useTankProfile, useUpsertTankProfile } from '@/hooks/useTankProfile'
import { useAgentSettings, useUpdateAgentSettings, useAgentContext, useAgentSkills } from '@/hooks/useAgent'
import type { ProbeConfig, OutletConfig, NotificationTarget, SystemLogLine, Device, DeviceSuggestion, InputCategory } from '@/api/types'
import type { TankSection, TankType, TankProfileInput } from '@/api/client'

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

type Tab = 'dashboard' | 'devices' | 'probes' | 'outlets' | 'tokens' | 'notifications' | 'backup' | 'log' | 'system' | 'tank' | 'agent'

const tabs: { key: Tab; label: string }[] = [
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

// --- Shared drag sensors ---

function useDragSensors() {
  return useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 20 } }),
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

const binaryCategoryOptions: { value: InputCategory; label: string }[] = [
  { value: 'switch', label: 'Switch' },
  { value: 'fluid', label: 'Fluid Sensor' },
  { value: 'alarm', label: 'Alarm' },
  { value: 'virtual', label: 'Virtual' },
]

function getCategoryIcon(category: InputCategory): { Icon: typeof ToggleLeft; color: string; bg: string } {
  switch (category) {
    case 'fluid': return { Icon: Droplets, color: 'text-amber-400', bg: 'bg-amber-400/10' }
    case 'alarm': return { Icon: AlertTriangle, color: 'text-tertiary', bg: 'bg-tertiary/10' }
    case 'virtual': return { Icon: Activity, color: 'text-primary', bg: 'bg-primary/10' }
    default: return { Icon: ToggleLeft, color: 'text-on-surface-dim', bg: 'bg-surface-container-high' }
  }
}

function CategorySelect({ value, onSave }: { value: InputCategory; onSave: (val: string) => void }) {
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
  onDisplayModeChange,
}: {
  item: DashboardItem
  displayName: string
  onRemove: (id: number) => void
  onLabelChange?: (id: number, label: string) => void
  onDisplayModeChange?: (id: number, mode: 'normal' | 'compact') => void
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
          : item.item_type === 'feed_mode'
            ? 'bg-primary/10 text-primary'
            : item.item_type === 'measurement'
              ? 'bg-secondary/10 text-secondary'
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
      {item.item_type !== 'separator' && onDisplayModeChange && (
        <button
          onClick={() =>
            onDisplayModeChange(item.id, item.display_mode === 'compact' ? 'normal' : 'compact')
          }
          title={item.display_mode === 'compact' ? 'Switch to normal' : 'Switch to compact'}
          className={cn(
            'p-1.5 rounded-lg transition-fluid cursor-pointer',
            item.display_mode === 'compact'
              ? 'text-primary bg-primary/10 hover:bg-primary/20'
              : 'text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container-high',
          )}
        >
          {item.display_mode === 'compact' ? <LayoutGrid size={16} /> : <LayoutList size={16} />}
        </button>
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
  const { data: measParamsData } = useMeasurementParameters()
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
    if (item.item_type === 'feed_mode') return 'Feed Mode'
    const ref = item.reference_id ?? ''
    switch (item.item_type) {
      case 'probe': return probeNameMap.get(ref) ?? ref
      case 'outlet': return outletNameMap.get(ref) ?? ref
      case 'device': return deviceNameMap.get(ref) ?? ref
      case 'measurement': return ref
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
  const availableMeasurements = (measParamsData?.parameters ?? []).filter((p) => !onDashboard.has(`measurement:${p.name}`))

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
        display_mode: i.display_mode,
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
      display_mode: 'normal',
    })
    setShowPicker(false)
  }

  function handleAddSeparator() {
    addMutation.mutate({
      item_type: 'separator',
      reference_id: null,
      label: 'New Section',
      display_mode: 'normal',
    })
  }

  function handleLabelChange(id: number, label: string) {
    // Update the label inline by replacing the full layout.
    const updated = items.map((i) => ({
      item_type: i.item_type,
      reference_id: i.reference_id,
      label: i.id === id ? label : i.label,
      display_mode: i.display_mode,
    }))
    replaceMutation.mutate(updated)
  }

  function handleDisplayModeChange(id: number, mode: 'normal' | 'compact') {
    const updated = items.map((i) => ({
      item_type: i.item_type,
      reference_id: i.reference_id,
      label: i.label,
      display_mode: i.id === id ? mode : i.display_mode,
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
          {availableMeasurements.length > 0 && (
            <div>
              <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1">Chemistry</p>
              <div className="flex flex-wrap gap-1.5">
                {availableMeasurements.map((p) => (
                  <button
                    key={p.name}
                    onClick={() => handleAdd('measurement', p.name)}
                    className="px-2.5 py-1 text-xs rounded-full bg-secondary/10 text-secondary hover:bg-secondary/20 transition-fluid cursor-pointer"
                  >
                    {p.name}
                  </button>
                ))}
              </div>
            </div>
          )}
          {!onDashboard.has('feed_mode:feed') && (
            <div>
              <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1">Controller</p>
              <div className="flex flex-wrap gap-1.5">
                <button
                  onClick={() => handleAdd('feed_mode', 'feed')}
                  className="px-2.5 py-1 text-xs rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-fluid cursor-pointer"
                >
                  Feed Mode
                </button>
              </div>
            </div>
          )}
          {availableProbes.length === 0 && availableOutlets.length === 0 && availableDevices.length === 0 && availableMeasurements.length === 0 && onDashboard.has('feed_mode:feed') && (
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
                onDisplayModeChange={item.item_type !== 'separator' ? handleDisplayModeChange : undefined}
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
  probes: { name: string; display_name: string; type: string; unit: string; input_category?: string; on_label?: string | null; off_label?: string | null; is_binary?: boolean; hidden?: boolean }[] | undefined,
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
      input_category: (existing?.input_category ?? p.input_category ?? 'probe') as ProbeConfig['input_category'],
      on_label: existing?.on_label ?? p.on_label ?? null,
      off_label: existing?.off_label ?? p.off_label ?? null,
      ok_value: existing?.ok_value ?? null,
      is_binary: existing?.is_binary ?? p.is_binary ?? false,
      hidden: existing?.hidden ?? p.hidden ?? false,
    }
  })
}

function ProbesTab() {
  const { data: configData, isLoading: configsLoading } = useProbeConfigs()
  const { data: probesData, isLoading: probesLoading } = useProbes()
  const updateMutation = useUpdateProbeConfig()

  const isLoading = configsLoading || probesLoading
  const allItems = useMergedProbeConfigs(probesData?.probes, configData?.configs)

  // Split into analog probes and binary inputs.
  const analogProbes = allItems.filter((c) => !c.is_binary)
  const binaryInputs = allItems.filter((c) => c.is_binary)

  // Map probe name → raw type so we can identify mis-categorized digital probes.
  const probeTypeMap = new Map((probesData?.probes ?? []).map((p) => [p.name, p.type]))

  function handleUpdate(name: string, field: keyof ProbeConfig, raw: string | boolean | null) {
    const numericFields: (keyof ProbeConfig)[] = ['min_normal', 'max_normal', 'min_warning', 'max_warning']
    const value =
      typeof raw === 'boolean' ? raw
      : numericFields.includes(field) ? (raw === '' || raw === null ? null : Number(raw))
      : raw
    updateMutation.mutate({ name, config: { [field]: value } })
  }

  if (isLoading) return <LoadingState label="Loading probe configs..." />

  if (allItems.length === 0) {
    return (
      <EmptyState
        icon={<SettingsIcon size={32} />}
        message="No probes found. Probes will appear here after the first poll."
      />
    )
  }

  return (
    <div className="space-y-6 pt-5">
      {analogProbes.length > 0 && (
        <div>
          <p className="text-xs text-on-surface-dim uppercase tracking-widest font-semibold px-5 mb-2">Analog Probes</p>
          <div className="overflow-x-auto">
          <table className="w-full min-w-[750px]">
            <thead>
              <tr className="bg-surface-container-high">
                {['Probe', 'Display Name', 'Unit', 'Min Normal', 'Max Normal', 'Min Warn', 'Max Warn', ''].map((h) => (
                  <th key={h} className="text-left py-3.5 px-5 text-xs font-semibold text-on-surface-dim uppercase tracking-widest">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {analogProbes.map((c) => {
                const isDigital = probeTypeMap.get(c.probe_name) === 'digital'
                return (
                  <tr key={c.probe_name} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-2 px-5 text-sm font-medium text-on-surface">{c.probe_name}</td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <UnitSelect value={c.unit_override} onSave={(v) => handleUpdate(c.probe_name, 'unit_override', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.min_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_normal', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.max_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_normal', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.min_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_warning', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.max_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_warning', v)} />
                    </td>
                    <td className="py-2 px-5">
                      {isDigital && (
                        <button
                          onClick={() => handleUpdate(c.probe_name, 'is_binary', true)}
                          className="inline-flex items-center gap-1 text-xs text-on-surface-faint hover:text-primary transition-fluid whitespace-nowrap"
                          title="Move to Binary Inputs"
                        >
                          <ArrowRight size={11} />
                          <span>binary</span>
                        </button>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          </div>
        </div>
      )}

      {binaryInputs.length > 0 && (
        <div>
          <div className="flex items-baseline justify-between px-5 mb-2">
            <p className="text-xs text-on-surface-dim uppercase tracking-widest font-semibold">Binary Inputs</p>
            <p className="text-xs text-on-surface-faint">Click a cell to edit · Category controls dashboard icon &amp; alert behavior</p>
          </div>
          <div className="overflow-x-auto">
          <table className="w-full min-w-[640px]">
            <thead>
              <tr className="bg-surface-container-high">
                {['Input', 'Category', 'Display Name', 'On Label', 'Off Label', 'Hidden'].map((h) => (
                  <th key={h} className="text-left py-3.5 px-5 text-xs font-semibold text-on-surface-dim uppercase tracking-widest">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {binaryInputs.filter((c) => c.probe_name !== '').map((c) => {
                const { Icon, color, bg } = getCategoryIcon(c.input_category)
                return (
                  <tr key={c.probe_name} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-2 px-5">
                      <div className="flex items-center gap-2.5">
                        <div className={cn('w-7 h-7 rounded-lg flex items-center justify-center shrink-0', bg)}>
                          <Icon size={13} className={color} />
                        </div>
                        <span className="text-sm font-medium text-on-surface">{c.probe_name}</span>
                      </div>
                    </td>
                    <td className="py-2 px-5">
                      <CategorySelect value={c.input_category} onSave={(v) => handleUpdate(c.probe_name, 'input_category', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.on_label ?? ''} onSave={(v) => handleUpdate(c.probe_name, 'on_label', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.off_label ?? ''} onSave={(v) => handleUpdate(c.probe_name, 'off_label', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <button
                        onClick={() => handleUpdate(c.probe_name, 'hidden', !c.hidden)}
                        className={cn(
                          'text-xs px-2 py-1 rounded-full transition-fluid',
                          c.hidden
                            ? 'bg-surface-container-high text-on-surface-faint'
                            : 'bg-secondary/15 text-secondary',
                        )}
                      >
                        {c.hidden ? 'Hidden' : 'Visible'}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          </div>
        </div>
      )}
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
  const updateOutletMutation = useUpdateOutletConfig()

  const isLoading = outletConfigsLoading || outletsLoading
  const items = useMergedOutletConfigs(outletsData?.outlets, outletConfigData?.configs)

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
          {items.map((item) => (
              <tr key={item.outlet_id} className="transition-fluid hover:bg-surface-container-high/50">
                <td className="py-2 px-4 text-sm font-medium text-on-surface">{item.outletName}</td>
                <td className="py-2 px-4">
                  <EditableCell value={item.display_name} onSave={(v) => handleUpdate(item.outlet_id, 'display_name', v)} />
                </td>
                <td className="py-2 px-4 text-xs text-on-surface-faint font-mono">{item.outlet_id}</td>
              </tr>
          ))}
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
// Tank Tab
// =============================================================================

const TANK_TYPE_LABELS: Record<TankType, string> = {
  reef: 'Reef',
  fowlr: 'Fish Only with Live Rock (FOWLR)',
  mixed: 'Mixed Reef',
  nano: 'Nano Reef',
  freshwater: 'Freshwater',
  other: 'Other',
}

function TankProfileForm({
  section,
  title,
}: {
  section: TankSection
  title: string
}) {
  const { data, isLoading } = useTankProfile()
  const upsert = useUpsertTankProfile()
  const profile = data?.[section]

  const [expanded, setExpanded] = useState(false)
  const [form, setForm] = useState<Partial<TankProfileInput>>({})
  const [saved, setSaved] = useState(false)

  // Populate form when profile loads
  useEffect(() => {
    if (profile) {
      setForm({
        display_name: profile.display_name ?? undefined,
        volume_gallons: profile.volume_gallons ?? undefined,
        length_in: profile.length_in ?? undefined,
        width_in: profile.width_in ?? undefined,
        height_in: profile.height_in ?? undefined,
        tank_type: profile.tank_type ?? undefined,
        manufacturer: profile.manufacturer ?? undefined,
        model: profile.model ?? undefined,
        setup_date: profile.setup_date ?? undefined,
        notes: profile.notes ?? undefined,
      })
    }
  }, [profile])

  function field(key: keyof TankProfileInput) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
      const val = e.target.value
      setForm(prev => ({ ...prev, [key]: val === '' ? undefined : val }))
    }
  }

  function numericField(key: keyof TankProfileInput) {
    return (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = parseFloat(e.target.value)
      setForm(prev => ({ ...prev, [key]: isNaN(val) ? undefined : val }))
    }
  }

  async function handleSave() {
    await upsert.mutateAsync({ section, data: form })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  if (isLoading) return <div className="py-3 text-xs text-on-surface-faint animate-pulse px-4">Loading...</div>

  const hasData = profile && (profile.display_name || profile.volume_gallons || profile.tank_type)

  return (
    <div className="rounded-2xl bg-surface-container-high/40 overflow-hidden">
      <button
        onClick={() => setExpanded(v => !v)}
        className="w-full flex items-center justify-between px-4 py-3 hover:bg-surface-container-high/60 transition-fluid"
      >
        <div className="flex items-center gap-3">
          <Waves size={16} className="text-primary/60" />
          <div className="text-left">
            <p className="text-sm font-semibold text-on-surface">{title}</p>
            {hasData && (
              <p className="text-xs text-on-surface-dim mt-0.5">
                {[profile.volume_gallons && `${profile.volume_gallons} gal`, profile.tank_type && TANK_TYPE_LABELS[profile.tank_type as TankType]].filter(Boolean).join(' · ')}
              </p>
            )}
          </div>
        </div>
        <span className={cn('text-on-surface-faint transition-fluid', expanded && 'rotate-180')}>▾</span>
      </button>

      {expanded && (
        <div className="px-4 pb-4 space-y-3 border-t border-surface-container-highest/30">
          <div className="pt-3 grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* Display name */}
            <div className="sm:col-span-2">
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Display Name</label>
              <input
                type="text"
                value={form.display_name ?? ''}
                onChange={field('display_name')}
                placeholder={title}
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
              />
            </div>

            {/* Tank type */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Type</label>
              <select
                value={form.tank_type ?? ''}
                onChange={field('tank_type')}
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
              >
                <option value="">Select type...</option>
                {(Object.keys(TANK_TYPE_LABELS) as TankType[]).map(t => (
                  <option key={t} value={t}>{TANK_TYPE_LABELS[t]}</option>
                ))}
              </select>
            </div>

            {/* Volume */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Volume (gallons)</label>
              <input
                type="number"
                min="0"
                step="0.1"
                value={form.volume_gallons ?? ''}
                onChange={numericField('volume_gallons')}
                placeholder="e.g. 75"
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
              />
            </div>

            {/* Dimensions */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Length (in)</label>
              <input type="number" min="0" step="0.5" value={form.length_in ?? ''} onChange={numericField('length_in')} placeholder="48" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Width (in)</label>
              <input type="number" min="0" step="0.5" value={form.width_in ?? ''} onChange={numericField('width_in')} placeholder="18" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Height (in)</label>
              <input type="number" min="0" step="0.5" value={form.height_in ?? ''} onChange={numericField('height_in')} placeholder="20" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>

            {/* Manufacturer & model */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Manufacturer</label>
              <input type="text" value={form.manufacturer ?? ''} onChange={field('manufacturer')} placeholder="e.g. Red Sea" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Model</label>
              <input type="text" value={form.model ?? ''} onChange={field('model')} placeholder="e.g. Reefer 350" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>

            {/* Setup date */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Setup Date</label>
              <input type="date" value={form.setup_date ?? ''} onChange={field('setup_date')} className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>

            {/* Notes */}
            <div className="sm:col-span-2">
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Notes</label>
              <textarea
                rows={2}
                value={form.notes ?? ''}
                onChange={field('notes')}
                placeholder="Any additional notes..."
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid resize-none"
              />
            </div>
          </div>

          <div className="flex justify-end">
            <button
              onClick={handleSave}
              disabled={upsert.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-xl bg-primary/10 text-primary text-sm font-medium hover:bg-primary/20 transition-fluid disabled:opacity-50"
            >
              {saved ? <><Check size={14} /> Saved</> : upsert.isPending ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function TankTab() {
  return (
    <div className="space-y-4 p-4">
      <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-1">Tank Profile</p>
      <TankProfileForm section="display" title="Display Tank" />
      <TankProfileForm section="sump" title="Sump" />
    </div>
  )
}

// =============================================================================
// Tokens Tab
// =============================================================================

function TokensTab() {
  const { data, isLoading } = useTokens()
  const createMutation = useCreateToken()
  const updateScopeMutation = useUpdateTokenScope()
  const revokeMutation = useRevokeToken()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [scope, setScope] = useState<'read' | 'control' | 'admin'>('admin')
  const [newToken, setNewToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [revokeConfirm, setRevokeConfirm] = useState<number | null>(null)

  const tokens = data?.tokens ?? []

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!label.trim()) return
    createMutation.mutate({ label: label.trim(), scope }, {
      onSuccess: (data) => {
        setNewToken(data.token)
        setLabel('')
        setScope('admin')
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
            className="flex-1 bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
            autoFocus
          />
          <select
            value={scope}
            onChange={(e) => setScope(e.target.value as 'read' | 'control' | 'admin')}
            className="bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid cursor-pointer"
          >
            <option value="admin">admin</option>
            <option value="control">control</option>
            <option value="read">read</option>
          </select>
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
              setScope('admin')
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
                {['ID', 'Label', 'Scope', 'Created', 'Last Used', ''].map((h) => (
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
                  <td className="py-3 px-4">
                    <select
                      value={t.scope}
                      onChange={(e) => updateScopeMutation.mutate({ id: t.id, scope: e.target.value })}
                      disabled={updateScopeMutation.isPending}
                      className="bg-surface-container-highest text-on-surface text-xs rounded-lg px-2 py-1 outline-none focus:ring-1 focus:ring-primary/30 cursor-pointer transition-fluid disabled:opacity-50"
                    >
                      <option value="admin">admin</option>
                      <option value="control">control</option>
                      <option value="read">read</option>
                    </select>
                  </td>
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
            className="flex-1 min-w-32 bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
            autoFocus
          />
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://ntfy.sh/your-topic"
            className="flex-[3] min-w-48 bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
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
// System Tab
// =============================================================================

function EventBusPanel() {
  const { data, isLoading } = useEventBusStats()
  const subs = data?.subscribers ?? []

  if (isLoading) return <LoadingState label="Loading bus stats..." />

  return (
    <div>
      <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Event Bus</p>
      <div className="bg-surface-container-high/40 rounded-xl overflow-hidden">
        {subs.length === 0 && (
          <p className="px-4 py-3 text-sm text-on-surface-faint">No subscribers registered.</p>
        )}
        {subs.map((s, i) => (
          <div
            key={s.name}
            className={cn(
              'px-4 py-3',
              i < subs.length - 1 ? 'border-b border-surface-container-highest/30' : '',
            )}
          >
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm text-on-surface font-medium font-mono">{s.name}</span>
              {s.dropped > 0 && (
                <span className="text-xs px-1.5 py-0.5 rounded-full bg-tertiary/10 text-tertiary font-medium">
                  {s.dropped} dropped
                </span>
              )}
              {s.dropped === 0 && (
                <span className="text-xs px-1.5 py-0.5 rounded-full bg-secondary/10 text-secondary font-medium">
                  healthy
                </span>
              )}
            </div>
            <div className="flex gap-4">
              <div>
                <p className="text-xs text-on-surface-faint">Queue</p>
                <p className="text-sm text-on-surface font-mono">{s.queue_depth}</p>
              </div>
              <div>
                <p className="text-xs text-on-surface-faint">Published</p>
                <p className="text-sm text-on-surface font-mono">{s.published.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-xs text-on-surface-faint">Dropped</p>
                <p className={cn('text-sm font-mono', s.dropped > 0 ? 'text-tertiary' : 'text-on-surface')}>{s.dropped}</p>
              </div>
            </div>
            {s.last_error && (
              <p className="text-xs text-tertiary mt-1 truncate" title={s.last_error}>{s.last_error}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

function SystemTab() {
  const { data, isLoading, refetch, isFetching } = useSystemStatus()

  if (isLoading) return <LoadingState label="Loading system status..." />

  if (!data) {
    return (
      <EmptyState
        icon={<Activity size={32} />}
        message="System status unavailable. The API server may not be reachable."
      />
    )
  }

  const stat = (label: string, value: string) => (
    <div className="py-3 px-4 flex items-center justify-between">
      <span className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">{label}</span>
      <span className="text-sm text-on-surface font-mono">{value}</span>
    </div>
  )

  return (
    <div className="space-y-6 p-4">
      {/* Poller status banner */}
      <div
        className={cn(
          'flex items-center gap-3 rounded-2xl px-4 py-3',
          data.poller.poll_ok ? 'bg-secondary/10' : 'bg-tertiary/10',
        )}
      >
        <span
          className={cn(
            'h-2.5 w-2.5 rounded-full shrink-0',
            data.poller.poll_ok ? 'bg-secondary animate-bio-pulse' : 'bg-tertiary',
          )}
        />
        <div className="flex-1">
          <p className={cn('text-sm font-semibold', data.poller.poll_ok ? 'text-secondary' : 'text-tertiary')}>
            {data.poller.poll_ok ? 'Poller running' : 'Poller degraded'}
          </p>
          <p className="text-xs text-on-surface-dim mt-0.5">
            Last poll {relativeTime(data.poller.last_poll_ts)} · every {data.poller.poll_interval_seconds}s
          </p>
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="p-1.5 rounded-lg text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high transition-fluid cursor-pointer disabled:opacity-50"
        >
          <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
        </button>
      </div>

      {/* Controller */}
      <div>
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Controller</p>
        <div className="bg-surface-container-high/40 rounded-xl divide-y divide-surface-container-highest/50">
          {stat('Serial', data.controller.serial)}
          {stat('Firmware', data.controller.firmware)}
          {stat('Hardware', data.controller.hardware)}
        </div>
      </div>

      {/* Databases */}
      <div>
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Databases</p>
        <div className="bg-surface-container-high/40 rounded-xl divide-y divide-surface-container-highest/50">
          {stat('Telemetry (DuckDB)', formatBytes(data.db.duckdb_size_bytes))}
          {stat('App state (SQLite)', formatBytes(data.db.sqlite_size_bytes))}
        </div>
      </div>

      {/* Event Bus */}
      <EventBusPanel />

      {/* Interface */}
      <div>
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-4 mb-1">Interface</p>
        <div className="bg-surface-container-high/40 rounded-xl">
          <div className="py-3 px-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Droplets size={16} className="text-primary/60" />
              <div>
                <p className="text-sm text-on-surface font-medium">Floating bubbles</p>
                <p className="text-xs text-on-surface-faint">Decorative background animation</p>
              </div>
            </div>
            <button
              onClick={() => { setBubblesEnabled(!getBubblesEnabled()); window.location.reload() }}
              className={cn(
                'px-3 py-1 rounded-full text-xs font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
                getBubblesEnabled()
                  ? 'bg-primary/20 text-primary'
                  : 'bg-surface-container-highest text-on-surface-faint',
              )}
            >
              {getBubblesEnabled() ? 'On' : 'Off'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// =============================================================================
// Agent Tab
// =============================================================================

function AgentTab() {
  const { data: settings, isLoading: settingsLoading } = useAgentSettings()
  const { data: skillsData, isLoading: skillsLoading } = useAgentSkills()
  const { data: contextData, isLoading: contextLoading, refetch: refetchContext } = useAgentContext()
  const updateSettings = useUpdateAgentSettings()
  const createToken = useCreateToken()
  const [showContext, setShowContext] = useState(false)
  const [copied, setCopied] = useState(false)
  const [personaCopied, setPersonaCopied] = useState(false)
  const [connectorToken, setConnectorToken] = useState<string | null>(null)
  const [connectorCopied, setConnectorCopied] = useState(false)

  if (settingsLoading || skillsLoading) return <LoadingState label="Loading agent settings..." />
  if (!settings) return null

  const skills = skillsData?.skills ?? []
  const context = contextData?.context ?? ''

  function handleToneChange(tone: string) {
    updateSettings.mutate({ tone: tone as 'analytical' | 'casual' | 'terse' })
  }

  function handleProductLineChange(line: string) {
    updateSettings.mutate({ dosing_product_line: line as 'brs_pharma' | 'red_sea' | 'tropic_marin' | 'generic' | 'none' })
  }

  function handleVolumeChange(val: string) {
    const n = parseFloat(val)
    if (val === '') {
      updateSettings.mutate({ net_volume_gallons: null })
    } else if (!isNaN(n) && n > 0) {
      updateSettings.mutate({ net_volume_gallons: n })
    }
  }

  function handleGuardrailsChange(val: string) {
    updateSettings.mutate({ custom_guardrails: val || null })
  }

  function handleSkillToggle(name: string, enabled: boolean) {
    const allNames = skills.map((s) => s.name)
    // Expand empty list (opt-out "all enabled") to explicit list before mutating,
    // otherwise filtering an empty array always stays empty.
    const current =
      settings!.enabled_skills && settings!.enabled_skills.length > 0
        ? settings!.enabled_skills
        : allNames
    let next: string[]
    if (enabled) {
      next = current.includes(name) ? current : [...current, name]
    } else {
      next = current.filter((n) => n !== name)
    }
    // Collapse back to empty when all skills are on (opt-out model)
    if (next.length === allNames.length) next = []
    updateSettings.mutate({ enabled_skills: next })
  }

  function isSkillEnabled(name: string): boolean {
    if (!settings!.enabled_skills || settings!.enabled_skills.length === 0) return true
    return settings!.enabled_skills.includes(name)
  }

  async function handleCopyContext() {
    await refetchContext()
    if (context) {
      navigator.clipboard.writeText(context).then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      })
    }
  }

  async function handleCopyPersona() {
    const { data } = await refetchContext()
    const text = data?.context ?? context
    if (text) {
      navigator.clipboard.writeText(text).then(() => {
        setPersonaCopied(true)
        setTimeout(() => setPersonaCopied(false), 2000)
      })
    }
  }

  function handleCreateConnectorToken() {
    createToken.mutate({ label: 'claude-mobile', scope: 'control' }, {
      onSuccess: (data) => setConnectorToken(data.token),
    })
  }

  function handleCopyConnectorToken(token: string) {
    navigator.clipboard.writeText(token).then(() => {
      setConnectorCopied(true)
      setTimeout(() => setConnectorCopied(false), 2000)
    })
  }

  const selectClass = 'w-full bg-surface-container-highest text-on-surface text-base rounded-xl px-3 py-2.5 outline-none focus:ring-2 focus:ring-primary/40 transition-fluid'
  const labelClass = 'text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1.5 block'

  return (
    <div className="space-y-6 p-4">

      {/* Header */}
      <div className="flex items-center gap-3 px-1 pt-1">
        <div className="h-8 w-8 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
          <Bot size={16} className="text-primary" />
        </div>
        <div>
          <p className="text-sm font-semibold text-on-surface">AI Assistant Persona</p>
          <p className="text-xs text-on-surface-faint">Configure how Claude responds to your tank data</p>
        </div>
      </div>

      {/* Persona controls */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">

        {/* Tone */}
        <div>
          <label className={labelClass}>Tone</label>
          <select
            value={settings.tone}
            onChange={(e) => handleToneChange(e.target.value)}
            className={selectClass}
          >
            <option value="analytical">Analytical — precise, data-driven</option>
            <option value="casual">Casual — conversational, friendly</option>
            <option value="terse">Terse — minimal, just the facts</option>
          </select>
        </div>

        {/* Dosing product line */}
        <div>
          <label className={labelClass}>Dosing Product Line</label>
          <select
            value={settings.dosing_product_line}
            onChange={(e) => handleProductLineChange(e.target.value)}
            className={selectClass}
          >
            <option value="generic">Generic (brand-neutral)</option>
            <option value="brs_pharma">BRS Pharma</option>
            <option value="red_sea">Red Sea</option>
            <option value="tropic_marin">Tropic Marin</option>
            <option value="none">None (no dosing advice)</option>
          </select>
        </div>

        {/* Net volume override */}
        <div>
          <label className={labelClass}>Net Volume Override (gallons)</label>
          <input
            type="number"
            min={1}
            step={0.5}
            placeholder="Derived from tank profile"
            defaultValue={settings.net_volume_gallons ?? ''}
            onBlur={(e) => handleVolumeChange(e.target.value)}
            className={cn(selectClass, 'placeholder:text-on-surface-faint/50')}
          />
          <p className="text-xs text-on-surface-faint mt-1">
            Leave blank to use the sum of display + sump volumes
          </p>
        </div>
      </div>

      {/* Custom guardrails */}
      <div>
        <label className={labelClass}>Custom Guardrails</label>
        <textarea
          rows={3}
          placeholder="e.g. Never recommend Alk above 10 dKH for this tank."
          defaultValue={settings.custom_guardrails ?? ''}
          onBlur={(e) => handleGuardrailsChange(e.target.value)}
          className={cn(selectClass, 'resize-y placeholder:text-on-surface-faint/50')}
        />
        <p className="text-xs text-on-surface-faint mt-1">
          Appended verbatim to every context block sent to Claude
        </p>
      </div>

      {/* Skills */}
      <div>
        <p className={labelClass}>Skills</p>
        <p className="text-xs text-on-surface-faint mb-3">
          Toggle skills to include or exclude them from installs. All skills are enabled by default.
        </p>
        {skillsLoading ? (
          <LoadingState label="Loading skills..." />
        ) : skills.length === 0 ? (
          <EmptyState icon={<Sparkles size={24} />} message="No skills available." />
        ) : (
          <div className="space-y-1">
            {skills.map((skill) => {
              const on = isSkillEnabled(skill.name)
              return (
                <div
                  key={skill.name}
                  className="flex items-start gap-3 px-3 py-3 rounded-xl bg-surface-container-high/40 hover:bg-surface-container-high/70 transition-fluid"
                >
                  <button
                    onClick={() => handleSkillToggle(skill.name, !on)}
                    className={cn(
                      'mt-0.5 h-4 w-4 rounded-full shrink-0 border-2 transition-fluid',
                      on
                        ? 'bg-primary border-primary'
                        : 'bg-transparent border-on-surface-faint/40',
                    )}
                    aria-label={on ? `Disable ${skill.name}` : `Enable ${skill.name}`}
                  />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-on-surface leading-snug">{skill.name}</p>
                    <p className="text-xs text-on-surface-faint leading-snug mt-0.5">{skill.description}</p>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Install instructions */}
      <div className="rounded-2xl bg-surface-container-high/40 p-4 space-y-2">
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">Install Skills</p>
        <p className="text-sm text-on-surface-dim">
          Run this command to install enabled skills into your Claude Code skills directory:
        </p>
        <div className="flex items-center gap-2 bg-surface-container-highest rounded-xl px-3 py-2 font-mono text-xs text-primary">
          <span className="flex-1 select-all">symbiont skills install</span>
          <span className="text-on-surface-faint">→ ~/.claude/skills/symbiont/</span>
        </div>
        <p className="text-xs text-on-surface-faint">
          Claude Code auto-discovers skills in that directory. Re-run after toggling skills or updating Symbiont.
        </p>
      </div>

      {/* claude.ai Connector */}
      <div className="rounded-2xl bg-surface-container-high/40 p-4 space-y-4">
        <div className="flex items-center gap-2">
          <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">claude.ai Connector</p>
        </div>
        <p className="text-sm text-on-surface-dim">
          Add Symbiont as a remote MCP connector in a claude.ai Project for mobile access.
          The MCP endpoint is at <code className="text-primary font-mono text-xs">/api/mcp</code> on your Symbiont host.
        </p>

        {/* Step 1: Persona */}
        <div className="space-y-1.5">
          <p className="text-xs text-on-surface-dim font-medium">1. Copy your agent persona</p>
          <p className="text-xs text-on-surface-faint">Paste into the Project&apos;s Custom Instructions so Claude knows your tank without calling tools every turn.</p>
          <button
            onClick={handleCopyPersona}
            className="flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-highest hover:bg-surface-container-high transition-fluid cursor-pointer"
          >
            {personaCopied ? <Check size={13} className="text-secondary" /> : <Copy size={13} />}
            {personaCopied ? 'Persona copied!' : 'Copy persona for claude.ai'}
          </button>
        </div>

        {/* Step 2: Connector token */}
        <div className="space-y-1.5">
          <p className="text-xs text-on-surface-dim font-medium">2. Create a connector token</p>
          <p className="text-xs text-on-surface-faint">Creates a <code className="text-primary font-mono">control</code>-scoped token labeled <code className="text-primary font-mono">claude-mobile</code>. Use it as the Bearer token in the claude.ai connector settings.</p>
          {connectorToken ? (
            <div className="space-y-2">
              <div className="flex items-center gap-2 bg-surface-container-highest rounded-xl px-3 py-2">
                <code className="flex-1 text-xs text-secondary font-mono break-all">{connectorToken}</code>
                <button
                  onClick={() => handleCopyConnectorToken(connectorToken)}
                  className="p-1 rounded-lg text-on-surface-faint hover:text-secondary hover:bg-secondary/10 transition-fluid cursor-pointer shrink-0"
                >
                  {connectorCopied ? <Check size={13} /> : <Copy size={13} />}
                </button>
              </div>
              <p className="text-xs text-tertiary">Save this token — it won&apos;t be shown again.</p>
              <button
                onClick={() => setConnectorToken(null)}
                className="text-xs text-on-surface-faint hover:text-on-surface transition-fluid cursor-pointer"
              >
                Dismiss
              </button>
            </div>
          ) : (
            <button
              onClick={handleCreateConnectorToken}
              disabled={createToken.isPending}
              className="flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
            >
              <Plus size={13} />
              Create connector token
            </button>
          )}
        </div>

        {/* Step 3: Walkthrough */}
        <div className="space-y-1.5">
          <p className="text-xs text-on-surface-dim font-medium">3. Add connector in claude.ai</p>
          <ol className="text-xs text-on-surface-faint space-y-1 list-decimal list-inside">
            <li>Open a Project in claude.ai → <span className="text-on-surface-dim">Project settings → Connectors</span></li>
            <li>Click <span className="text-on-surface-dim">Add connector → Custom MCP server</span></li>
            <li>Enter your Symbiont URL and the token above</li>
            <li>Paste the persona into <span className="text-on-surface-dim">Custom Instructions</span></li>
          </ol>
          <p className="text-xs text-on-surface-faint pt-1">
            Need a public URL?{' '}
            <span className="text-on-surface-dim">Use Tailscale Funnel or Cloudflare Tunnel — see <code className="text-primary font-mono">docs/deployment-remote-mcp.md</code>.</span>
          </p>
        </div>
      </div>

      {/* Context preview */}
      <div>
        <button
          onClick={() => { setShowContext((v) => !v); if (!showContext) refetchContext() }}
          className="flex items-center gap-2 text-xs text-on-surface-dim hover:text-on-surface transition-fluid"
        >
          {showContext ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          <span className="uppercase tracking-widest font-medium">Preview assembled context</span>
        </button>

        {showContext && (
          <div className="mt-3 space-y-2">
            <div className="flex justify-end">
              <button
                onClick={handleCopyContext}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid"
              >
                {copied ? <Check size={13} className="text-secondary" /> : <Copy size={13} />}
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
            {contextLoading ? (
              <LoadingState label="Building context..." />
            ) : (
              <pre className="bg-surface-container-highest rounded-xl p-4 text-xs text-on-surface-dim overflow-auto max-h-96 whitespace-pre-wrap leading-relaxed font-mono">
                {context || '(empty)'}
              </pre>
            )}
          </div>
        )}
      </div>
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

      {/* Desktop: pill tabs */}
      <div className="hidden lg:flex flex-wrap gap-1.5">
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

      {/* Mobile/tablet: select dropdown */}
      <div className="lg:hidden relative">
        <select
          value={activeTab}
          onChange={(e) => setActiveTab(e.target.value as Tab)}
          className="w-full appearance-none bg-surface-container-high text-on-surface text-base font-semibold uppercase tracking-wider rounded-xl pl-4 pr-10 py-3 outline-none focus:ring-1 focus:ring-primary/50 cursor-pointer"
        >
          {tabs.map((tab) => (
            <option key={tab.key} value={tab.key}>
              {tab.label}
            </option>
          ))}
        </select>
        <div className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-on-surface-dim">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M2 4l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </div>
      </div>

      <div className="bg-surface-container rounded-2xl overflow-hidden">
        {activeTab === 'dashboard' && <DashboardTab />}
        {activeTab === 'devices' && <DevicesTab />}
        {activeTab === 'tank' && <TankTab />}
        {activeTab === 'probes' && <ProbesTab />}
        {activeTab === 'outlets' && <OutletsTab />}
        {activeTab === 'agent' && <AgentTab />}
        {activeTab === 'tokens' && <TokensTab />}
        {activeTab === 'notifications' && <NotificationsTab />}
        {activeTab === 'backup' && <BackupTab />}
        {activeTab === 'log' && <SystemLogTab />}
        {activeTab === 'system' && <SystemTab />}
      </div>
    </div>
  )
}

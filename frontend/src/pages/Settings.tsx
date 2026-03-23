import { useState, useRef, useEffect } from 'react'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
  arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Settings as SettingsIcon, Plus, Trash2, Copy, Check, Download, RefreshCw, EyeOff, Eye, GripVertical } from 'lucide-react'
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
import type { ProbeConfig, OutletConfig } from '@/api/types'

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

type Tab = 'dashboard' | 'probes' | 'outlets' | 'tokens' | 'backup'

const tabs: { key: Tab; label: string }[] = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'probes', label: 'Probes' },
  { key: 'outlets', label: 'Outlets' },
  { key: 'tokens', label: 'Tokens' },
  { key: 'backup', label: 'Backup' },
]

// --- Shared drag sensors ---

function useDragSensors() {
  return useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 5 } }),
  )
}

// --- Drag handle ---

function DragHandle({ listeners, attributes }: { listeners?: Record<string, Function>; attributes?: Record<string, unknown> }) {
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
// Dashboard Tab — drag-and-drop ordering for probe cards and outlet cards
// =============================================================================

/** A sortable item in the dashboard layout. */
type DashboardLayoutItem =
  | { kind: 'probe'; id: string; displayName: string; type: string; hidden: boolean; displayOrder: number }
  | { kind: 'power-pair'; id: string; displayName: string; wattsProbe: string; ampsProbe: string; hidden: boolean; displayOrder: number }
  | { kind: 'outlet'; id: string; displayName: string; outletId: string; hidden: boolean; displayOrder: number }

function SortableDashboardRow({
  item,
  onVisibilityToggle,
}: {
  item: DashboardLayoutItem
  onVisibilityToggle: (item: DashboardLayoutItem) => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  const typeLabel = item.kind === 'probe'
    ? item.type
    : item.kind === 'power-pair'
      ? 'Power'
      : 'Outlet'

  const typeBadgeStyle = item.kind === 'outlet'
    ? 'bg-primary/10 text-primary'
    : item.kind === 'power-pair'
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
        {typeLabel}
      </span>
      <span className="flex-1 text-sm font-medium text-on-surface truncate">
        {item.displayName}
      </span>
      <button
        onClick={() => onVisibilityToggle(item)}
        className={cn(
          'p-1.5 rounded-lg transition-fluid cursor-pointer',
          item.hidden
            ? 'text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high'
            : 'text-primary hover:bg-primary/10',
        )}
      >
        {item.hidden ? <EyeOff size={16} /> : <Eye size={16} />}
      </button>
    </div>
  )
}

function useProbeLayoutItems(
  probes: { name: string; display_name: string; type: string }[] | undefined,
  probeConfigs: ProbeConfig[] | undefined,
): DashboardLayoutItem[] {
  if (!probes) return []

  const probeCfgMap = new Map((probeConfigs ?? []).map((c) => [c.probe_name, c]))

  // Detect power pairs
  const wattsBases = new Map<string, string>()
  const ampsBases = new Map<string, string>()
  for (const p of probes) {
    if (p.type === 'pwr' && p.name.endsWith('W')) wattsBases.set(p.name.slice(0, -1), p.name)
    if (p.type === 'Amps' && p.name.endsWith('A')) ampsBases.set(p.name.slice(0, -1), p.name)
  }
  const pairedBases = new Set<string>()
  for (const base of wattsBases.keys()) {
    if (ampsBases.has(base)) pairedBases.add(base)
  }
  const pairedProbeNames = new Set<string>()
  for (const base of pairedBases) {
    pairedProbeNames.add(wattsBases.get(base)!)
    pairedProbeNames.add(ampsBases.get(base)!)
  }

  const items: DashboardLayoutItem[] = []

  // Add power pairs
  for (const base of pairedBases) {
    const wattsName = wattsBases.get(base)!
    const ampsName = ampsBases.get(base)!
    const wattsCfg = probeCfgMap.get(wattsName)
    items.push({
      kind: 'power-pair',
      id: `pair:${base}`,
      displayName: wattsCfg?.display_name ?? base,
      wattsProbe: wattsName,
      ampsProbe: ampsName,
      hidden: wattsCfg?.hidden ?? false,
      displayOrder: wattsCfg?.display_order ?? 0,
    })
  }

  // Add standalone probes
  for (const p of probes) {
    if (pairedProbeNames.has(p.name)) continue
    const cfg = probeCfgMap.get(p.name)
    items.push({
      kind: 'probe',
      id: `probe:${p.name}`,
      displayName: cfg?.display_name ?? p.display_name ?? p.name,
      type: p.type,
      hidden: cfg?.hidden ?? false,
      displayOrder: cfg?.display_order ?? 0,
    })
  }

  return sortByOrder(items)
}

function useOutletLayoutItems(
  outlets: { id: string; name: string; display_name: string; type: string }[] | undefined,
  outletConfigs: OutletConfig[] | undefined,
): DashboardLayoutItem[] {
  if (!outlets) return []
  const outletCfgMap = new Map((outletConfigs ?? []).map((c) => [c.outlet_id, c]))
  const items: DashboardLayoutItem[] = []

  for (const o of outlets) {
    if (o.type !== 'outlet' && o.type !== 'virtual') continue
    const cfg = outletCfgMap.get(o.id)
    items.push({
      kind: 'outlet',
      id: `outlet:${o.id}`,
      displayName: cfg?.display_name ?? o.display_name ?? o.name,
      outletId: o.id,
      hidden: cfg?.hidden ?? false,
      displayOrder: cfg?.display_order ?? 0,
    })
  }

  return sortByOrder(items)
}

function sortByOrder(items: DashboardLayoutItem[]): DashboardLayoutItem[] {
  return [...items].sort((a, b) => {
    if (a.displayOrder === 0 && b.displayOrder === 0) {
      return a.displayName.localeCompare(b.displayName)
    }
    if (a.displayOrder === 0) return 1
    if (b.displayOrder === 0) return -1
    return a.displayOrder - b.displayOrder
  })
}

/** A single sortable section with a label, drag list, and visibility toggles. */
function SortableSection({
  label,
  items,
  onDragEnd,
  onVisibilityToggle,
  sensors,
}: {
  label: string
  items: DashboardLayoutItem[]
  onDragEnd: (event: DragEndEvent) => void
  onVisibilityToggle: (item: DashboardLayoutItem) => void
  sensors: ReturnType<typeof useDragSensors>
}) {
  return (
    <div>
      <div className="px-4 py-2.5 bg-surface-container-high/50">
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">
          {label}
        </p>
      </div>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
        <SortableContext items={items.map((c) => c.id)} strategy={verticalListSortingStrategy}>
          {items.map((item) => (
            <SortableDashboardRow
              key={item.id}
              item={item}
              onVisibilityToggle={onVisibilityToggle}
            />
          ))}
        </SortableContext>
      </DndContext>
    </div>
  )
}

function DashboardTab() {
  const { data: probesData, isLoading: probesLoading } = useProbes()
  const { data: probeConfigData, isLoading: probeConfigsLoading } = useProbeConfigs()
  const { data: outletsData, isLoading: outletsLoading } = useOutlets()
  const { data: outletConfigData, isLoading: outletConfigsLoading } = useOutletConfigs()
  const updateProbeMutation = useUpdateProbeConfig()
  const updateOutletMutation = useUpdateOutletConfig()
  const sensors = useDragSensors()

  const isLoading = probesLoading || probeConfigsLoading || outletsLoading || outletConfigsLoading

  const mergedProbes = useProbeLayoutItems(probesData?.probes, probeConfigData?.configs)
  const mergedOutlets = useOutletLayoutItems(outletsData?.outlets, outletConfigData?.configs)

  const [localProbeOrder, setLocalProbeOrder] = useState<DashboardLayoutItem[] | null>(null)
  const [localOutletOrder, setLocalOutletOrder] = useState<DashboardLayoutItem[] | null>(null)
  const probeItems = localProbeOrder ?? mergedProbes
  const outletItems = localOutletOrder ?? mergedOutlets

  function saveProbeOrder(reordered: DashboardLayoutItem[]) {
    reordered.forEach((item, i) => {
      const order = i + 1
      if (item.kind === 'probe') {
        const name = item.id.replace('probe:', '')
        updateProbeMutation.mutate({ name, config: { display_order: order } })
      } else if (item.kind === 'power-pair') {
        updateProbeMutation.mutate({ name: item.wattsProbe, config: { display_order: order } })
        updateProbeMutation.mutate({ name: item.ampsProbe, config: { display_order: order } })
      }
    })
  }

  function saveOutletOrder(reordered: DashboardLayoutItem[]) {
    reordered.forEach((item, i) => {
      if (item.kind === 'outlet') {
        updateOutletMutation.mutate({ id: item.outletId, config: { display_order: i + 1 } })
      }
    })
  }

  function handleVisibilityToggle(item: DashboardLayoutItem) {
    const newHidden = !item.hidden
    if (item.kind === 'probe') {
      const name = item.id.replace('probe:', '')
      updateProbeMutation.mutate({ name, config: { hidden: newHidden } })
    } else if (item.kind === 'power-pair') {
      updateProbeMutation.mutate({ name: item.wattsProbe, config: { hidden: newHidden } })
      updateProbeMutation.mutate({ name: item.ampsProbe, config: { hidden: newHidden } })
    } else if (item.kind === 'outlet') {
      updateOutletMutation.mutate({ id: item.outletId, config: { hidden: newHidden } })
    }
  }

  function handleProbeDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) { setLocalProbeOrder(null); return }
    const oldIndex = probeItems.findIndex((c) => c.id === active.id)
    const newIndex = probeItems.findIndex((c) => c.id === over.id)
    const reordered = arrayMove(probeItems, oldIndex, newIndex)
    setLocalProbeOrder(reordered)
    saveProbeOrder(reordered)
    setTimeout(() => setLocalProbeOrder(null), 500)
  }

  function handleOutletDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) { setLocalOutletOrder(null); return }
    const oldIndex = outletItems.findIndex((c) => c.id === active.id)
    const newIndex = outletItems.findIndex((c) => c.id === over.id)
    const reordered = arrayMove(outletItems, oldIndex, newIndex)
    setLocalOutletOrder(reordered)
    saveOutletOrder(reordered)
    setTimeout(() => setLocalOutletOrder(null), 500)
  }

  if (isLoading) return <LoadingState label="Loading dashboard layout..." />

  if (probeItems.length === 0 && outletItems.length === 0) {
    return (
      <EmptyState
        icon={<SettingsIcon size={32} />}
        message="No probes or outlets found. Items will appear here after the first poll."
      />
    )
  }

  return (
    <div>
      <div className="px-4 py-3 bg-surface-container-high/30">
        <p className="text-xs text-on-surface-faint">
          Drag to reorder cards within each section. Toggle visibility with the eye icon.
        </p>
      </div>
      {probeItems.length > 0 && (
        <SortableSection
          label="Telemetry Cards"
          items={probeItems}
          onDragEnd={handleProbeDragEnd}
          onVisibilityToggle={handleVisibilityToggle}
          sensors={sensors}
        />
      )}
      {outletItems.length > 0 && (
        <SortableSection
          label="Controls"
          items={outletItems}
          onDragEnd={handleOutletDragEnd}
          onVisibilityToggle={handleVisibilityToggle}
          sensors={sensors}
        />
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

  // Detect power pairs to exclude (managed via Outlets tab)
  const wattsBases = new Set<string>()
  const ampsBases = new Set<string>()
  for (const p of probes) {
    if (p.type === 'pwr' && p.name.endsWith('W')) wattsBases.add(p.name.slice(0, -1))
    if (p.type === 'Amps' && p.name.endsWith('A')) ampsBases.add(p.name.slice(0, -1))
  }
  const pairedNames = new Set<string>()
  for (const base of wattsBases) {
    if (ampsBases.has(base)) {
      pairedNames.add(base + 'W')
      pairedNames.add(base + 'A')
    }
  }

  return probes
    .filter((p) => !pairedNames.has(p.name))
    .map((p) => {
      const existing = configMap.get(p.name)
      return {
        probe_name: p.name,
        display_name: existing?.display_name ?? p.display_name ?? p.name,
        unit_override: existing?.unit_override ?? p.unit ?? '',
        display_order: existing?.display_order ?? 0,
        min_normal: existing?.min_normal ?? null,
        max_normal: existing?.max_normal ?? null,
        min_warning: existing?.min_warning ?? null,
        max_warning: existing?.max_warning ?? null,
        hidden: existing?.hidden ?? false,
      }
    })
}

function ProbesTab() {
  const { data: configData, isLoading: configsLoading } = useProbeConfigs()
  const { data: probesData, isLoading: probesLoading } = useProbes()
  const updateMutation = useUpdateProbeConfig()

  const isLoading = configsLoading || probesLoading
  const items = useMergedProbeConfigs(probesData?.probes, configData?.configs)

  function handleUpdate(name: string, field: keyof ProbeConfig, raw: string) {
    const numericFields: (keyof ProbeConfig)[] = [
      'display_order', 'min_normal', 'max_normal', 'min_warning', 'max_warning',
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
          {items.map((c) => (
            <tr key={c.probe_name} className="transition-fluid hover:bg-surface-container-high/50">
              <td className="py-2 px-4 text-sm font-medium text-on-surface">{c.probe_name}</td>
              <td className="py-2 px-4">
                <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
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
          ))}
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
        display_order: existing?.display_order ?? 0,
        icon: existing?.icon ?? '',
        hidden: existing?.hidden ?? false,
        outletName: o.name,
      }
    })
}

function OutletsTab() {
  const { data: outletConfigData, isLoading: outletConfigsLoading } = useOutletConfigs()
  const { data: outletsData, isLoading: outletsLoading } = useOutlets()
  const { data: probesData } = useProbes()
  const { data: probeConfigData } = useProbeConfigs()
  const updateOutletMutation = useUpdateOutletConfig()
  const updateProbeMutation = useUpdateProbeConfig()

  const isLoading = outletConfigsLoading || outletsLoading
  const items = useMergedOutletConfigs(outletsData?.outlets, outletConfigData?.configs)

  // Build power probe lookup by outlet name
  const probes = probesData?.probes ?? []
  const wattsMap = new Map<string, { name: string; value: number }>()
  const ampsMap = new Map<string, { name: string; value: number }>()
  for (const p of probes) {
    if (p.type === 'pwr' && p.name.endsWith('W')) {
      wattsMap.set(p.name.slice(0, -1), { name: p.name, value: p.value })
    } else if (p.type === 'Amps' && p.name.endsWith('A')) {
      ampsMap.set(p.name.slice(0, -1), { name: p.name, value: p.value })
    }
  }
  const probeConfigMap = new Map((probeConfigData?.configs ?? []).map((c) => [c.probe_name, c]))

  function handleUpdate(id: string, field: keyof OutletConfig, raw: string) {
    const value = field === 'display_order'
      ? raw === '' ? 0 : Number(raw)
      : field === 'hidden'
        ? raw === 'true'
        : raw
    updateOutletMutation.mutate({ id, config: { [field]: value } })
  }

  function handleDisplayNameUpdate(item: OutletConfig & { outletName: string }, newName: string) {
    handleUpdate(item.outlet_id, 'display_name', newName)
    // Sync to linked power probes
    const watts = wattsMap.get(item.outletName)
    const amps = ampsMap.get(item.outletName)
    if (watts) updateProbeMutation.mutate({ name: watts.name, config: { display_name: newName } })
    if (amps) updateProbeMutation.mutate({ name: amps.name, config: { display_name: newName } })
  }

  function handlePowerThreshold(outletName: string, field: keyof ProbeConfig, raw: string) {
    const value = raw === '' ? null : Number(raw)
    const watts = wattsMap.get(outletName)
    const amps = ampsMap.get(outletName)
    if (watts) updateProbeMutation.mutate({ name: watts.name, config: { [field]: value } })
    if (amps) updateProbeMutation.mutate({ name: amps.name, config: { [field]: value } })
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
            {['Outlet', 'Display Name', 'Power', 'Max Watts', 'ID'].map((h) => (
              <th key={h} className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const watts = wattsMap.get(item.outletName)
            const amps = ampsMap.get(item.outletName)
            const wattsCfg = watts ? probeConfigMap.get(watts.name) : undefined
            return (
              <tr key={item.outlet_id} className="transition-fluid hover:bg-surface-container-high/50">
                <td className="py-2 px-4 text-sm font-medium text-on-surface">{item.outletName}</td>
                <td className="py-2 px-4">
                  <EditableCell value={item.display_name} onSave={(v) => handleDisplayNameUpdate(item, v)} />
                </td>
                <td className="py-2 px-4 text-sm">
                  {watts && amps ? (
                    <span className="text-on-surface font-mono">
                      {watts.value.toFixed(1)}W / {amps.value.toFixed(2)}A
                    </span>
                  ) : (
                    <span className="text-on-surface-faint">—</span>
                  )}
                </td>
                <td className="py-2 px-4 text-sm">
                  {watts ? (
                    <EditableCell
                      value={wattsCfg?.max_warning ?? null}
                      type="number"
                      onSave={(v) => handlePowerThreshold(item.outletName, 'max_warning', v)}
                    />
                  ) : (
                    <span className="text-on-surface-faint">—</span>
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

      <div className="flex gap-1.5">
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
        {activeTab === 'probes' && <ProbesTab />}
        {activeTab === 'outlets' && <OutletsTab />}
        {activeTab === 'tokens' && <TokensTab />}
        {activeTab === 'backup' && <BackupTab />}
      </div>
    </div>
  )
}

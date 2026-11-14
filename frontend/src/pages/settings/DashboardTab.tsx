import { useState } from 'react'
import { cn } from '@/lib/utils'
import { DndContext,closestCenter, type DragEndEvent } from '@dnd-kit/core'
import { SortableContext,verticalListSortingStrategy,useSortable,arrayMove } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Settings as SettingsIcon,Plus,Trash2,LayoutGrid,LayoutList } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets } from '@/hooks/useOutlets'
import { useMeasurementParameters } from '@/hooks/useMeasurements'
import { useDevices } from '@/hooks/useDevices'
import { Button, Chip, IconButton, LoadingState, EmptyState, labelClass, type ChipTone } from '@/components/ui'
import { ConfirmButton } from '@/components/ConfirmButton'
import { useDragSensors,DragHandle,EditableCell } from './_shared'

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

  const typeTone: ChipTone =
    item.item_type === 'outlet' || item.item_type === 'feed_mode'
      ? 'primary'
      : item.item_type === 'device'
        ? 'critical'
        : item.item_type === 'separator'
          ? 'warning'
          : 'secondary'

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
      <Chip tone={typeTone} className="shrink-0">{item.item_type}</Chip>
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
        <IconButton
          icon={item.display_mode === 'compact' ? LayoutGrid : LayoutList}
          size={16}
          label={item.display_mode === 'compact' ? 'Switch to normal' : 'Switch to compact'}
          onClick={() =>
            onDisplayModeChange(item.id, item.display_mode === 'compact' ? 'normal' : 'compact')
          }
          className={item.display_mode === 'compact' ? 'text-primary bg-primary/10 hover:bg-primary/20' : undefined}
        />
      )}
      <ConfirmButton
        onConfirm={() => onRemove(item.id)}
        title="Remove from dashboard"
        className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
      >
        <Trash2 size={16} />
      </ConfirmButton>
    </div>
  )
}

export default function DashboardTab() {
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
      <div className="px-4 py-3 bg-surface-container-high/30 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-on-surface-faint">
          Drag to reorder. Remove items with the trash icon.
        </p>
        <div className="flex gap-2">
          <Button size="sm" onClick={handleAddSeparator} className="flex-1 sm:flex-none">
            + Separator
          </Button>
          <Button variant="tonal" size="sm" icon={Plus} onClick={() => setShowPicker(!showPicker)} className="flex-1 sm:flex-none">
            Add Item
          </Button>
        </div>
      </div>

      {/* Picker */}
      {showPicker && (
        <div className="px-4 py-3 bg-surface-container-high/20 space-y-3">
          {availableProbes.length > 0 && (
            <div>
              <p className={`${labelClass} mb-1`}>Probes</p>
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
              <p className={`${labelClass} mb-1`}>Outlets</p>
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
              <p className={`${labelClass} mb-1`}>Devices</p>
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
              <p className={`${labelClass} mb-1`}>Chemistry</p>
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
              <p className={`${labelClass} mb-1`}>Controller</p>
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


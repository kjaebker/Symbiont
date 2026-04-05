import { useState, useRef, useEffect } from 'react'
import { Network, Edit2, Eye, Save, X } from 'lucide-react'
import { useSystemLayout, useSaveSystemLayout } from '@/hooks/useSystemLayout'
import SystemCanvas, { type SystemCanvasHandle } from '@/components/system/SystemCanvas'
import ElementPalette from '@/components/system/ElementPalette'
import { cn } from '@/lib/utils'
import type { SystemLayout, SystemNode } from '@/api/types'

type Mode = 'view' | 'edit'

interface LayerVisibility {
  water: boolean
  electrical: boolean
  sensor: boolean
}

export default function System() {
  const { data, isLoading } = useSystemLayout()
  const saveLayout = useSaveSystemLayout()
  const [mode, setMode] = useState<Mode>('view')
  const [layers, setLayers] = useState<LayerVisibility>({
    water: true,
    electrical: true,
    sensor: true,
  })
  const [snapToGrid, setSnapToGrid] = useState(true)
  const [editingLayout, setEditingLayout] = useState<SystemLayout | null>(null)
  const [hasUnsaved, setHasUnsaved] = useState(false)
  const canvasRef = useRef<SystemCanvasHandle | null>(null)

  const serverLayout = data?.layout ?? null

  // Warn on unload if unsaved
  useEffect(() => {
    if (!hasUnsaved) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [hasUnsaved])

  function enterEdit() {
    setEditingLayout(serverLayout ? JSON.parse(JSON.stringify(serverLayout)) : null)
    setHasUnsaved(false)
    setMode('edit')
  }

  function cancelEdit() {
    setEditingLayout(null)
    setHasUnsaved(false)
    setMode('view')
  }

  async function handleSave(layout: SystemLayout) {
    await saveLayout.mutateAsync(layout)
    setHasUnsaved(false)
    setMode('view')
    setEditingLayout(null)
  }

  function toggleLayer(layer: keyof LayerVisibility) {
    setLayers((prev) => ({ ...prev, [layer]: !prev[layer] }))
  }

  function handleDrop(event: React.DragEvent, position: { x: number; y: number }) {
    const raw = event.dataTransfer.getData('application/symbiont-node')
    if (!raw) return
    try {
      const item = JSON.parse(raw)
      const newNode: SystemNode = {
        id: `${item.nodeType}-${Date.now()}`,
        type: item.nodeType,
        position,
        data: {
          label: item.label,
          subtype: item.subtype,
          deviceId: item.deviceId,
          probeName: item.probeName,
        },
        ...(item.nodeType === 'container' ? { size: { width: 240, height: 160 } } : {}),
      }
      setEditingLayout((prev) => {
        const base = prev ?? makeEmptyLayout()
        return { ...base, nodes: [...base.nodes, newNode] }
      })
      setHasUnsaved(true)
    } catch {
      // ignore bad drag data
    }
  }

  const displayLayout = mode === 'edit' ? editingLayout : serverLayout

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center text-on-surface-dim">
        Loading...
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-4 py-3 bg-surface-container-low shrink-0 flex-wrap">
        <div className="flex items-center gap-2 mr-2">
          <Network size={18} className="text-primary" />
          <span className="text-sm font-semibold text-on-surface tracking-wide">System Layout</span>
        </div>

        {/* Mode toggle */}
        <div className="flex items-center rounded-xl bg-surface-container p-1 gap-1">
          <button
            onClick={() => mode !== 'view' ? cancelEdit() : undefined}
            disabled={mode === 'view'}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-fluid',
              mode === 'view'
                ? 'bg-surface-container-high text-primary shadow-glow-primary'
                : 'text-on-surface-dim hover:text-on-surface',
            )}
          >
            <Eye size={14} />
            View
          </button>
          <button
            onClick={() => mode !== 'edit' ? enterEdit() : undefined}
            disabled={mode === 'edit'}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-fluid',
              mode === 'edit'
                ? 'bg-surface-container-high text-secondary shadow-glow-secondary'
                : 'text-on-surface-dim hover:text-on-surface',
            )}
          >
            <Edit2 size={14} />
            Edit
          </button>
        </div>

        {/* Layer toggles */}
        <div className="flex items-center gap-2 ml-2">
          <button
            onClick={() => toggleLayer('water')}
            className={cn(
              'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-fluid',
              layers.water
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container text-on-surface-dim',
            )}
          >
            <span className="h-1.5 w-1.5 rounded-full bg-current" />
            Water
          </button>
          <button
            onClick={() => toggleLayer('electrical')}
            className={cn(
              'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-fluid',
              layers.electrical
                ? 'bg-amber-500/20 text-amber-400'
                : 'bg-surface-container text-on-surface-dim',
            )}
          >
            <span className="h-1.5 w-1.5 rounded-full bg-current" />
            Electrical
          </button>
          <button
            onClick={() => toggleLayer('sensor')}
            className={cn(
              'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-fluid',
              layers.sensor
                ? 'bg-secondary/20 text-secondary'
                : 'bg-surface-container text-on-surface-dim',
            )}
          >
            <span className="h-1.5 w-1.5 rounded-full bg-current" />
            Sensors
          </button>
        </div>

        {/* Edit mode controls */}
        {mode === 'edit' && (
          <>
            <button
              onClick={() => setSnapToGrid((v) => !v)}
              className={cn(
                'ml-1 px-2.5 py-1 rounded-full text-xs font-medium transition-fluid',
                snapToGrid
                  ? 'bg-surface-container-high text-on-surface'
                  : 'bg-surface-container text-on-surface-dim',
              )}
            >
              Snap
            </button>

            <div className="ml-auto flex items-center gap-2">
              <button
                onClick={cancelEdit}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-surface-container text-on-surface-dim hover:text-on-surface transition-fluid"
              >
                <X size={13} />
                Cancel
              </button>
              <button
                onClick={() => {
                  const layout = canvasRef.current?.getLayout(snapToGrid)
                  if (layout) handleSave(layout)
                }}
                disabled={saveLayout.isPending}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-secondary/20 text-secondary hover:bg-secondary/30 transition-fluid"
              >
                <Save size={13} />
                {saveLayout.isPending ? 'Saving…' : 'Save'}
              </button>
            </div>
          </>
        )}
      </div>

      {/* Canvas area */}
      <div className="flex flex-1 overflow-hidden">
        {mode === 'edit' && <ElementPalette />}

        <div
          className={cn(
            'flex-1 relative',
            !layers.water && 'hide-layer-water',
            !layers.electrical && 'hide-layer-electrical',
            !layers.sensor && 'hide-layer-sensor',
          )}
        >
          {displayLayout === null && mode === 'view' ? (
            <EmptyState onEdit={enterEdit} />
          ) : (
            <SystemCanvas
              layout={displayLayout}
              mode={mode}
              snapToGrid={snapToGrid}
              canvasRef={canvasRef}
              onDrop={mode === 'edit' ? handleDrop : undefined}
            />
          )}
        </div>
      </div>
    </div>
  )
}

function makeEmptyLayout(): SystemLayout {
  return {
    version: 1,
    viewport: { x: 0, y: 0, zoom: 1 },
    settings: { snapToGrid: true, gridSize: 20 },
    nodes: [],
    edges: [],
  }
}

function EmptyState({ onEdit }: { onEdit: () => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-6 text-center px-6">
      <div className="rounded-2xl bg-surface-container p-6 glass">
        <Network size={48} className="text-primary mx-auto mb-4 opacity-60" />
        <h2 className="text-lg font-semibold text-on-surface mb-2">No layout yet</h2>
        <p className="text-sm text-on-surface-dim max-w-xs mb-6">
          Build a visual map of your aquarium system — containers, devices, probes, and the
          connections between them.
        </p>
        <button
          onClick={onEdit}
          className="px-4 py-2 rounded-xl bg-primary/20 text-primary text-sm font-medium transition-fluid hover:bg-primary/30"
        >
          Start building
        </button>
      </div>
    </div>
  )
}

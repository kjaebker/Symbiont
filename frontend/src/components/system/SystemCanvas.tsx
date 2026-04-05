import { useCallback, useRef, forwardRef, useImperativeHandle } from 'react'
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  MiniMap,
  Controls,
  addEdge,
  useNodesState,
  useEdgesState,
  useReactFlow,
  type Connection,
  type Node,
  type Edge,
  type NodeTypes,
  type EdgeTypes,
  MarkerType,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { SystemDataProvider } from './SystemDataProvider'
import ContainerNode from './nodes/ContainerNode'
import DeviceNode from './nodes/DeviceNode'
import ProbeNode from './nodes/ProbeNode'
import PowerBarNode from './nodes/PowerBarNode'
import TPortNode from './nodes/TPortNode'
import ValveNode from './nodes/ValveNode'
import WaterFlowEdge from './edges/WaterFlowEdge'
import ElectricalEdge from './edges/ElectricalEdge'
import SensorEdge from './edges/SensorEdge'
import type { SystemLayout, SystemNode, SystemEdge, SystemEdgeLayer } from '@/api/types'

const nodeTypes: NodeTypes = {
  container: ContainerNode,
  device: DeviceNode,
  probe: ProbeNode,
  powerbar: PowerBarNode,
  tport: TPortNode,
  valve: ValveNode,
}

const edgeTypes: EdgeTypes = {
  water: WaterFlowEdge,
  electrical: ElectricalEdge,
  sensor: SensorEdge,
}

function inferEdgeLayer(
  sourceType: string | undefined,
  targetType: string | undefined,
): SystemEdgeLayer {
  if (sourceType === 'powerbar' || targetType === 'powerbar') return 'electrical'
  if (sourceType === 'probe' || targetType === 'probe') return 'sensor'
  return 'water'
}

function toFlowNodes(nodes: SystemNode[]): Node[] {
  return nodes.map((n) => ({
    id: n.id,
    type: n.type,
    position: n.position,
    data: n.data,
    parentId: n.parentId,
    extent: n.parentId ? ('parent' as const) : undefined,
    style: n.size ? { width: n.size.width, height: n.size.height } : undefined,
  }))
}

function toFlowEdges(edges: SystemEdge[], mode: 'view' | 'edit'): Edge[] {
  return edges.map((e) => ({
    id: e.id,
    source: e.source,
    sourceHandle: e.sourceHandle,
    target: e.target,
    targetHandle: e.targetHandle,
    type: e.layer,
    data: { ...e.data, mode },
    markerEnd: e.layer === 'water' ? { type: MarkerType.ArrowClosed, color: 'var(--color-primary)' } : undefined,
    className: `layer-${e.layer}`,
  }))
}

function fromFlowNodes(nodes: Node[]): SystemNode[] {
  return nodes.map((n) => ({
    id: n.id,
    type: n.type as SystemNode['type'],
    position: n.position,
    size: n.style?.width != null
      ? { width: n.style.width as number, height: n.style.height as number }
      : undefined,
    parentId: n.parentId,
    data: n.data as SystemNode['data'],
  }))
}

function fromFlowEdges(edges: Edge[]): SystemEdge[] {
  return edges.map((e) => ({
    id: e.id,
    source: e.source,
    sourceHandle: e.sourceHandle ?? undefined,
    target: e.target,
    targetHandle: e.targetHandle ?? undefined,
    layer: e.type as SystemEdgeLayer,
    data: e.data as SystemEdge['data'],
  }))
}

export interface SystemCanvasHandle {
  getLayout: (snapToGrid: boolean) => SystemLayout
}

interface SystemCanvasProps {
  layout: SystemLayout | null
  mode: 'view' | 'edit'
  snapToGrid: boolean
}

const SystemCanvasInner = forwardRef<SystemCanvasHandle, SystemCanvasProps>(
  function SystemCanvasInner({ layout, mode, snapToGrid }, ref) {
    const initialNodes = layout ? toFlowNodes(layout.nodes) : []
    const initialEdges = layout ? toFlowEdges(layout.edges, mode) : []

    const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
    const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)
    const reactFlowWrapper = useRef<HTMLDivElement>(null)
    const { getViewport, screenToFlowPosition } = useReactFlow()

    // Expose getLayout() to the parent (System.tsx save button)
    useImperativeHandle(ref, () => ({
      getLayout: (snap: boolean): SystemLayout => {
        const viewport = getViewport()
        return {
          version: 1,
          viewport,
          settings: { snapToGrid: snap, gridSize: 20 },
          nodes: fromFlowNodes(nodes),
          edges: fromFlowEdges(edges),
        }
      },
    }))

    const onConnect = useCallback(
      (connection: Connection) => {
        const sourceNode = nodes.find((n) => n.id === connection.source)
        const targetNode = nodes.find((n) => n.id === connection.target)
        const layer = inferEdgeLayer(sourceNode?.type, targetNode?.type)
        const newEdge: Edge = {
          ...connection,
          id: `e-${connection.source}-${connection.target}-${Date.now()}`,
          type: layer,
          data: { mode },
          markerEnd:
            layer === 'water'
              ? { type: MarkerType.ArrowClosed, color: 'var(--color-primary)' }
              : undefined,
          className: `layer-${layer}`,
        }
        setEdges((eds) => addEdge(newEdge, eds))
      },
      [nodes, mode, setEdges],
    )

    const handleDragOver = useCallback((event: React.DragEvent) => {
      event.preventDefault()
      event.dataTransfer.dropEffect = 'move'
    }, [])

    const handleDrop = useCallback(
      (event: React.DragEvent) => {
        event.preventDefault()
        const raw = event.dataTransfer.getData('application/symbiont-node')
        if (!raw) return
        let item: Record<string, unknown>
        try {
          item = JSON.parse(raw)
        } catch {
          return
        }
        // Convert screen coordinates to React Flow canvas coordinates
        const position = screenToFlowPosition({ x: event.clientX, y: event.clientY })
        const newNode: Node = {
          id: `${item.nodeType}-${Date.now()}`,
          type: item.nodeType as string,
          position,
          data: {
            label: item.label,
            subtype: item.subtype,
            deviceId: item.deviceId,
            probeName: item.probeName,
          },
          ...(item.nodeType === 'container'
            ? { style: { width: 240, height: 160 } }
            : {}),
        }
        setNodes((nds) => [...nds, newNode])
      },
      [screenToFlowPosition, setNodes],
    )

    // Inject mode into edge data so particles animate in view mode
    const edgesWithMode = edges.map((e) => ({
      ...e,
      data: { ...e.data, mode },
    }))

    const isEditMode = mode === 'edit'

    return (
      <div ref={reactFlowWrapper} className="w-full h-full">
        <ReactFlow
          nodes={nodes}
          edges={edgesWithMode}
          onNodesChange={isEditMode ? onNodesChange : undefined}
          onEdgesChange={isEditMode ? onEdgesChange : undefined}
          onConnect={isEditMode ? onConnect : undefined}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          nodesDraggable={isEditMode}
          nodesConnectable={isEditMode}
          elementsSelectable={isEditMode}
          snapToGrid={isEditMode && snapToGrid}
          snapGrid={[20, 20]}
          deleteKeyCode={isEditMode ? 'Delete' : null}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          onDragOver={isEditMode ? handleDragOver : undefined}
          onDrop={isEditMode ? handleDrop : undefined}
          proOptions={{ hideAttribution: true }}
        >
          {isEditMode && (
            <Background
              variant={BackgroundVariant.Dots}
              gap={20}
              size={1}
            />
          )}
          <MiniMap
            style={{
              background: 'var(--color-surface-container)',
              border: 'none',
            }}
            nodeColor="var(--color-surface-container-highest)"
            maskColor="rgba(7, 13, 31, 0.7)"
          />
          <Controls
            showInteractive={false}
            className="!bg-surface-container-low !border-0 !rounded-xl overflow-hidden"
          />
        </ReactFlow>
      </div>
    )
  },
)

export default function SystemCanvas(props: SystemCanvasProps & { canvasRef?: React.Ref<SystemCanvasHandle> }) {
  const { canvasRef, layout, mode, snapToGrid } = props
  return (
    <ReactFlowProvider>
      <SystemDataProvider>
        <SystemCanvasInner ref={canvasRef} layout={layout} mode={mode} snapToGrid={snapToGrid} />
      </SystemDataProvider>
    </ReactFlowProvider>
  )
}

import { memo } from 'react'
import { Handle, Position, NodeResizer } from '@xyflow/react'
import { Box, Layers, Droplets } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ContainerNodeData {
  label: string
  subtype?: 'tank' | 'sump' | 'ato'
}

interface ContainerNodeProps {
  data: ContainerNodeData
  selected?: boolean
  // injected by React Flow when in edit mode (via nodesDraggable context)
  isConnectable?: boolean
  // mode is passed via data to avoid nodeProps typing issues
}

const subtypeIcon = {
  tank: Box,
  sump: Layers,
  ato: Droplets,
}

const subtypeLabel = {
  tank: 'Tank',
  sump: 'Sump',
  ato: 'ATO Reservoir',
}

export default memo(function ContainerNode({ data, selected, isConnectable }: ContainerNodeProps) {
  const Icon = subtypeIcon[data.subtype ?? 'tank'] ?? Box
  const typeLabel = subtypeLabel[data.subtype ?? 'tank'] ?? 'Container'

  return (
    <div
      className={cn(
        'relative w-full h-full rounded-2xl transition-fluid',
        'bg-surface-container/40 backdrop-blur-sm',
        selected
          ? 'outline outline-2 outline-primary/60 shadow-glow-primary'
          : 'outline outline-[1.5px] outline-white/[0.07]',
      )}
    >
      <NodeResizer
        minWidth={120}
        minHeight={80}
        isVisible={selected}
        lineClassName="!border-primary/40"
        handleClassName="!bg-surface-container-high !border-primary/60 !rounded-sm"
      />

      {/* Header */}
      <div className="flex items-center gap-1.5 px-3 pt-2.5 pb-1">
        <Icon size={13} className="text-on-surface-dim shrink-0" />
        <span className="text-[11px] font-medium text-on-surface truncate leading-none">
          {data.label}
        </span>
        <span className="ml-auto text-[10px] text-on-surface-faint uppercase tracking-widest shrink-0">
          {typeLabel}
        </span>
      </div>

      {/* Water connection handles on all four sides */}
      <Handle
        type="source"
        position={Position.Top}
        id="water-top"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full"
      />
      <Handle
        type="target"
        position={Position.Top}
        id="water-top-in"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full !left-[60%]"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="water-bottom"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full"
      />
      <Handle
        type="target"
        position={Position.Bottom}
        id="water-bottom-in"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full !left-[60%]"
      />
      <Handle
        type="source"
        position={Position.Left}
        id="water-left"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full"
      />
      <Handle
        type="target"
        position={Position.Left}
        id="water-left-in"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full !top-[60%]"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="water-right"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full"
      />
      <Handle
        type="target"
        position={Position.Right}
        id="water-right-in"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-3 !h-3 !rounded-full !top-[60%]"
      />
    </div>
  )
})

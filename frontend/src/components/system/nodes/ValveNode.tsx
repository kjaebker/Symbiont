import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { cn } from '@/lib/utils'

interface ValveNodeData {
  label?: string
  open?: boolean
}

interface ValveNodeProps {
  data: ValveNodeData
  selected?: boolean
  isConnectable?: boolean
}

export default memo(function ValveNode({ data, selected, isConnectable }: ValveNodeProps) {
  const isOpen = data.open !== false // default open

  return (
    <div
      className={cn(
        'relative flex flex-col items-center justify-center w-10 h-10 rounded-xl transition-fluid',
        'bg-surface-container-highest',
        selected && 'outline outline-2 outline-primary/60',
        isOpen ? 'shadow-glow-secondary' : '',
      )}
    >
      {/* Valve symbol — two triangles */}
      <svg viewBox="0 0 20 20" className={cn('w-5 h-5', isOpen ? 'text-secondary' : 'text-tertiary')}>
        <polygon points="2,4 10,10 2,16" fill="currentColor" opacity="0.8" />
        <polygon points="18,4 10,10 18,16" fill="currentColor" opacity="0.8" />
        <line x1="10" y1="2" x2="10" y2="18"
          stroke="currentColor" strokeWidth="1.5" opacity={isOpen ? '1' : '0.3'} />
      </svg>

      {/* 1 input, 1 output */}
      <Handle
        type="target"
        position={Position.Left}
        id="in"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-2.5 !h-2.5 !rounded-full"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="out"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-2.5 !h-2.5 !rounded-full"
      />
    </div>
  )
})

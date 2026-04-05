import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { cn } from '@/lib/utils'

interface TPortNodeProps {
  selected?: boolean
  isConnectable?: boolean
}

export default memo(function TPortNode({ selected, isConnectable }: TPortNodeProps) {
  return (
    <div
      className={cn(
        'relative flex items-center justify-center w-8 h-8 rounded-full transition-fluid',
        'bg-surface-container-highest',
        selected
          ? 'outline outline-2 outline-primary/60 shadow-glow-primary'
          : 'outline outline-[1.5px] outline-primary/30',
      )}
    >
      <div className="w-2.5 h-2.5 rounded-full bg-primary/60" />

      {/* 1 input on Left */}
      <Handle
        type="target"
        position={Position.Left}
        id="in"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-2.5 !h-2.5 !rounded-full"
      />
      {/* 2 outputs on Right */}
      <Handle
        type="source"
        position={Position.Right}
        id="out-top"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-2.5 !h-2.5 !rounded-full !top-[30%]"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="out-bottom"
        isConnectable={isConnectable}
        className="!bg-primary/70 !border-primary/40 !w-2.5 !h-2.5 !rounded-full !top-[70%]"
      />
    </div>
  )
})

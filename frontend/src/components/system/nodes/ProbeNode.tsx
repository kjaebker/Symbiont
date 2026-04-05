import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { Activity } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSystemData } from '../SystemDataProvider'

interface ProbeNodeData {
  label: string
  probeName?: string
}

interface ProbeNodeProps {
  data: ProbeNodeData
  selected?: boolean
  isConnectable?: boolean
}

const statusColor = {
  normal: 'text-secondary',
  warning: 'text-amber-400',
  critical: 'text-tertiary',
  unknown: 'text-on-surface-dim',
}

export default memo(function ProbeNode({ data, selected, isConnectable }: ProbeNodeProps) {
  const { probesByName } = useSystemData()
  const probe = data.probeName ? probesByName[data.probeName] : undefined

  return (
    <div
      className={cn(
        'relative px-3 py-2 rounded-xl min-w-[100px] transition-fluid',
        'bg-surface-container',
        selected && 'outline outline-2 outline-secondary/60',
      )}
    >
      <div className="flex items-center gap-1.5 mb-1">
        <Activity size={12} className="text-secondary shrink-0" />
        <span className="text-[11px] font-medium text-on-surface-dim truncate">
          {data.label}
        </span>
      </div>

      {probe ? (
        <div className={cn('text-sm font-bold leading-none', statusColor[probe.status])}>
          {probe.value.toFixed(2)}
          <span className="text-[10px] font-normal ml-1 text-on-surface-dim">{probe.unit}</span>
        </div>
      ) : (
        <div className="text-xs text-on-surface-faint">—</div>
      )}

      {/* Sensor output */}
      <Handle
        type="source"
        position={Position.Right}
        id="sensor-out"
        isConnectable={isConnectable}
        className="!bg-secondary/70 !border-secondary/40 !w-2.5 !h-2.5 !rounded-full"
      />
    </div>
  )
})

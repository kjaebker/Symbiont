import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { Cpu } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSystemData } from '../SystemDataProvider'

interface DeviceNodeData {
  label: string
  deviceId?: number
  subtype?: string
}

interface DeviceNodeProps {
  data: DeviceNodeData
  selected?: boolean
  isConnectable?: boolean
}

export default memo(function DeviceNode({ data, selected, isConnectable }: DeviceNodeProps) {
  const { outletsById, devicesById } = useSystemData()
  const device = data.deviceId != null ? devicesById[data.deviceId] : undefined
  const outlet = device?.outlet_id ? outletsById[device.outlet_id] : undefined
  const isOn = outlet ? ['ON', 'AON'].includes(outlet.state) : null

  return (
    <div
      className={cn(
        'relative px-3 py-2 rounded-xl min-w-[110px] transition-fluid',
        'bg-surface-container-high',
        selected && 'outline outline-2 outline-primary/60',
        isOn === true && 'shadow-glow-secondary',
        isOn === false && 'opacity-60',
      )}
    >
      <div className="flex items-center gap-2">
        <Cpu size={14} className={cn(
          'shrink-0',
          isOn === true ? 'text-secondary' : 'text-on-surface-dim',
        )} />
        <span className="text-xs font-medium text-on-surface truncate max-w-[120px]">
          {data.label}
        </span>
        {isOn !== null && (
          <span
            className={cn(
              'ml-auto h-1.5 w-1.5 rounded-full shrink-0',
              isOn ? 'bg-secondary' : 'bg-on-surface-faint',
            )}
          />
        )}
      </div>

      {/* Electrical input */}
      <Handle
        type="target"
        position={Position.Left}
        id="electrical-in"
        isConnectable={isConnectable}
        className="!bg-amber-500/70 !border-amber-500/40 !w-2.5 !h-2.5 !rounded-full !-left-1.5"
      />
      {/* Sensor input */}
      <Handle
        type="target"
        position={Position.Bottom}
        id="sensor-in"
        isConnectable={isConnectable}
        className="!bg-secondary/70 !border-secondary/40 !w-2.5 !h-2.5 !rounded-full"
      />
    </div>
  )
})

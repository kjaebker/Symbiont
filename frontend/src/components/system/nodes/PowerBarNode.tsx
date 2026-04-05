import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { Plug } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSystemData } from '../SystemDataProvider'
import type { PowerBarSlot } from '@/api/types'

interface PowerBarNodeData {
  label: string
  slots?: PowerBarSlot[]
}

interface PowerBarNodeProps {
  data: PowerBarNodeData
  selected?: boolean
  isConnectable?: boolean
}

export default memo(function PowerBarNode({ data, selected, isConnectable }: PowerBarNodeProps) {
  const { outletsById } = useSystemData()
  const slots = data.slots ?? []

  return (
    <div
      className={cn(
        'relative rounded-xl overflow-hidden transition-fluid',
        'bg-surface-container-high min-w-[160px]',
        selected && 'outline outline-2 outline-amber-500/60',
      )}
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2 bg-surface-container-highest">
        <Plug size={12} className="text-amber-400 shrink-0" />
        <span className="text-xs font-semibold text-on-surface truncate">{data.label}</span>
      </div>

      {/* Slots */}
      <div className="flex flex-col">
        {Array.from({ length: 8 }, (_, i) => i + 1).map((idx) => {
          const slot = slots.find((s) => s.index === idx)
          const outlet = slot ? outletsById[slot.outletId] : undefined
          const isOn = outlet ? ['ON', 'AON'].includes(outlet.state) : null

          return (
            <div
              key={idx}
              className="relative flex items-center gap-2 px-3 py-1.5"
            >
              <span className="text-[10px] text-on-surface-faint w-3 shrink-0">{idx}</span>
              <span className="text-[11px] text-on-surface-dim truncate flex-1">
                {outlet?.display_name ?? outlet?.name ?? (slot ? slot.outletId : '—')}
              </span>
              <span
                className={cn(
                  'h-1.5 w-1.5 rounded-full shrink-0',
                  isOn === true ? 'bg-secondary' : isOn === false ? 'bg-on-surface-faint' : 'bg-surface-container',
                )}
              />
              <Handle
                type="source"
                position={Position.Right}
                id={`slot-${idx}`}
                isConnectable={isConnectable}
                style={{ top: '50%' }}
                className="!bg-amber-500/70 !border-amber-500/40 !w-2 !h-2 !rounded-full !-right-1"
              />
            </div>
          )
        })}
      </div>
    </div>
  )
})

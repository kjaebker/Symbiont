import { memo } from 'react'
import { getSmoothStepPath, EdgeLabelRenderer, BaseEdge } from '@xyflow/react'
import type { EdgeProps } from '@xyflow/react'

interface ElectricalEdgeData {
  outletOn?: boolean
  wattage?: number
}

export default memo(function ElectricalEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  markerEnd,
}: EdgeProps<ElectricalEdgeData>) {
  const isOn = data?.outletOn !== false
  const midX = (sourceX + targetX) / 2
  const midY = (sourceY + targetY) / 2

  const [edgePath] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    borderRadius: 4,
  })

  return (
    <g className="layer-electrical">
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: '#f59e0b',
          strokeWidth: 2,
          strokeDasharray: isOn ? undefined : '6 4',
          strokeOpacity: isOn ? 0.9 : 0.45,
        }}
      />

      {data?.wattage != null && (
        <EdgeLabelRenderer>
          <div
            className="absolute text-[10px] text-amber-400 bg-surface-container px-1.5 py-0.5 rounded-full pointer-events-none"
            style={{ transform: `translate(-50%, -50%) translate(${midX}px,${midY}px)` }}
          >
            {data.wattage}W
          </div>
        </EdgeLabelRenderer>
      )}
    </g>
  )
})

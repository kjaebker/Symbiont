import { memo } from 'react'
import { getBezierPath, EdgeLabelRenderer, BaseEdge } from '@xyflow/react'
import type { EdgeProps } from '@xyflow/react'

interface SensorEdgeData {
  reading?: string
}

export default memo(function SensorEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps<SensorEdgeData>) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  return (
    <g className="layer-sensor">
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: 'var(--color-secondary)',
          strokeWidth: 1.5,
          strokeOpacity: 0.7,
        }}
      />

      {data?.reading != null && (
        <EdgeLabelRenderer>
          <div
            className="absolute text-[10px] text-secondary bg-surface-container px-1.5 py-0.5 rounded-full pointer-events-none"
            style={{ transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)` }}
          >
            {data.reading}
          </div>
        </EdgeLabelRenderer>
      )}
    </g>
  )
})

import { memo } from 'react'
import { getSmoothStepPath, EdgeLabelRenderer, BaseEdge } from '@xyflow/react'
import type { EdgeProps } from '@xyflow/react'

interface WaterFlowEdgeData {
  flowDirection?: 'forward' | 'reverse'
  mode?: 'view' | 'edit'
}

export default memo(function WaterFlowEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  markerEnd,
}: EdgeProps<WaterFlowEdgeData>) {
  const [edgePath] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    borderRadius: 4,
  })

  const isViewMode = data?.mode !== 'edit'

  return (
    <g className="layer-water">
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: 'var(--color-primary)',
          strokeWidth: 3,
          strokeOpacity: 0.85,
        }}
      />

      {/* Animated flow particles (view mode only) */}
      {isViewMode && (
        <>
          <circle r="3" fill="var(--color-primary)" opacity="0.9">
            <animateMotion
              dur="2.4s"
              repeatCount="indefinite"
              begin="0s"
              path={edgePath}
            >
            </animateMotion>
          </circle>
          <circle r="3" fill="var(--color-primary)" opacity="0.9">
            <animateMotion
              dur="2.4s"
              repeatCount="indefinite"
              begin="0.8s"
              path={edgePath}
            >
            </animateMotion>
          </circle>
          <circle r="3" fill="var(--color-primary)" opacity="0.9">
            <animateMotion
              dur="2.4s"
              repeatCount="indefinite"
              begin="1.6s"
              path={edgePath}
            >
            </animateMotion>
          </circle>
        </>
      )}
    </g>
  )
})

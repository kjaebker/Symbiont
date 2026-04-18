import { useEffect, useRef, useState, useCallback } from 'react'
import { RotateCcw, RotateCw, Check, X } from 'lucide-react'

interface ImageEditorProps {
  file: File
  onConfirm: (file: File) => void
  onCancel: () => void
}

interface Crop {
  x: number // 0–100 percent of canvas
  y: number
  w: number
  h: number
}

type HandleId = 'nw' | 'ne' | 'se' | 'sw' | 'move'

interface DragState {
  handle: HandleId
  startCx: number
  startCy: number
  startCrop: Crop
}

const MIN_PCT = 5

function clamp(v: number, lo: number, hi: number) {
  return Math.min(hi, Math.max(lo, v))
}

export function ImageEditor({ file, onConfirm, onCancel }: ImageEditorProps) {
  const [rotation, setRotation] = useState(0)
  const [crop, setCrop] = useState<Crop>({ x: 0, y: 0, w: 100, h: 100 })
  const [ready, setReady] = useState(false)
  const [processing, setProcessing] = useState(false)

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const bitmapRef = useRef<ImageBitmap | null>(null)
  const dragRef = useRef<DragState | null>(null)
  const cropRef = useRef<Crop>({ x: 0, y: 0, w: 100, h: 100 })
  const rotationRef = useRef(0)

  cropRef.current = crop
  rotationRef.current = rotation

  useEffect(() => {
    let cancelled = false
    createImageBitmap(file, { imageOrientation: 'from-image' }).then((bm) => {
      if (!cancelled) {
        bitmapRef.current = bm
        setReady(true)
      }
    })
    return () => {
      cancelled = true
      bitmapRef.current?.close()
      bitmapRef.current = null
    }
  }, [file])

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    const bm = bitmapRef.current
    if (!canvas || !bm) return

    const container = containerRef.current!
    const maxW = container.clientWidth
    const maxH = container.clientHeight
    if (maxW === 0 || maxH === 0) return

    const r = rotationRef.current
    const is90 = r === 90 || r === 270
    const nw = bm.width
    const nh = bm.height
    const rw = is90 ? nh : nw
    const rh = is90 ? nw : nh

    const scale = Math.min(maxW / rw, maxH / rh, 1)
    const cw = Math.round(rw * scale)
    const ch = Math.round(rh * scale)

    canvas.width = cw
    canvas.height = ch

    const ctx = canvas.getContext('2d')!

    // Draw rotated image
    ctx.save()
    ctx.translate(cw / 2, ch / 2)
    ctx.rotate((r * Math.PI) / 180)
    ctx.drawImage(bm, (-nw * scale) / 2, (-nh * scale) / 2, nw * scale, nh * scale)
    ctx.restore()

    // Crop overlay
    const c = cropRef.current
    const cx = (c.x / 100) * cw
    const cy = (c.y / 100) * ch
    const cropW = (c.w / 100) * cw
    const cropH = (c.h / 100) * ch

    ctx.fillStyle = 'rgba(0,0,0,0.55)'
    ctx.fillRect(0, 0, cw, cy)
    ctx.fillRect(0, cy + cropH, cw, ch - cy - cropH)
    ctx.fillRect(0, cy, cx, cropH)
    ctx.fillRect(cx + cropW, cy, cw - cx - cropW, cropH)

    // Rule of thirds
    ctx.strokeStyle = 'rgba(255,255,255,0.2)'
    ctx.lineWidth = 0.5
    for (let i = 1; i < 3; i++) {
      ctx.beginPath()
      ctx.moveTo(cx + (cropW * i) / 3, cy)
      ctx.lineTo(cx + (cropW * i) / 3, cy + cropH)
      ctx.stroke()
      ctx.beginPath()
      ctx.moveTo(cx, cy + (cropH * i) / 3)
      ctx.lineTo(cx + cropW, cy + (cropH * i) / 3)
      ctx.stroke()
    }

    // Crop border
    ctx.strokeStyle = 'rgba(255,255,255,0.85)'
    ctx.lineWidth = 1.5
    ctx.strokeRect(cx + 0.75, cy + 0.75, cropW - 1.5, cropH - 1.5)

    // Corner brackets
    const hl = Math.max(14, Math.min(24, cw * 0.04))
    ctx.strokeStyle = 'white'
    ctx.lineWidth = 2.5
    ctx.lineCap = 'round'
    // NW
    ctx.beginPath(); ctx.moveTo(cx + hl, cy); ctx.lineTo(cx, cy); ctx.lineTo(cx, cy + hl); ctx.stroke()
    // NE
    ctx.beginPath(); ctx.moveTo(cx + cropW - hl, cy); ctx.lineTo(cx + cropW, cy); ctx.lineTo(cx + cropW, cy + hl); ctx.stroke()
    // SE
    ctx.beginPath(); ctx.moveTo(cx + cropW, cy + cropH - hl); ctx.lineTo(cx + cropW, cy + cropH); ctx.lineTo(cx + cropW - hl, cy + cropH); ctx.stroke()
    // SW
    ctx.beginPath(); ctx.moveTo(cx, cy + cropH - hl); ctx.lineTo(cx, cy + cropH); ctx.lineTo(cx + hl, cy + cropH); ctx.stroke()
  }, [])

  useEffect(() => {
    if (ready) draw()
  }, [rotation, crop, ready, draw])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const ro = new ResizeObserver(() => { if (ready) draw() })
    ro.observe(container)
    return () => ro.disconnect()
  }, [draw, ready])

  function rotateLeft() {
    setRotation((r) => (r - 90 + 360) % 360)
    setCrop({ x: 0, y: 0, w: 100, h: 100 })
  }

  function rotateRight() {
    setRotation((r) => (r + 90) % 360)
    setCrop({ x: 0, y: 0, w: 100, h: 100 })
  }

  function canvasPct(e: React.PointerEvent): { cx: number; cy: number } {
    const canvas = canvasRef.current!
    const rect = canvas.getBoundingClientRect()
    return {
      cx: ((e.clientX - rect.left) / rect.width) * 100,
      cy: ((e.clientY - rect.top) / rect.height) * 100,
    }
  }

  function getHandle(cx: number, cy: number): HandleId | null {
    const c = cropRef.current
    const canvas = canvasRef.current!
    const hrX = (44 / canvas.getBoundingClientRect().width) * 100
    const hrY = (44 / canvas.getBoundingClientRect().height) * 100

    if (Math.abs(cx - c.x) < hrX && Math.abs(cy - c.y) < hrY) return 'nw'
    if (Math.abs(cx - (c.x + c.w)) < hrX && Math.abs(cy - c.y) < hrY) return 'ne'
    if (Math.abs(cx - (c.x + c.w)) < hrX && Math.abs(cy - (c.y + c.h)) < hrY) return 'se'
    if (Math.abs(cx - c.x) < hrX && Math.abs(cy - (c.y + c.h)) < hrY) return 'sw'

    if (cx >= c.x && cx <= c.x + c.w && cy >= c.y && cy <= c.y + c.h) return 'move'
    return null
  }

  function handlePointerDown(e: React.PointerEvent) {
    e.currentTarget.setPointerCapture(e.pointerId)
    const { cx, cy } = canvasPct(e)
    const handle = getHandle(cx, cy)
    if (!handle) return
    dragRef.current = { handle, startCx: cx, startCy: cy, startCrop: { ...cropRef.current } }
    e.preventDefault()
  }

  function handlePointerMove(e: React.PointerEvent) {
    const d = dragRef.current
    if (!d) return

    const { cx, cy } = canvasPct(e)
    const dx = cx - d.startCx
    const dy = cy - d.startCy
    const sc = d.startCrop
    let { x, y, w, h } = sc

    switch (d.handle) {
      case 'move':
        x = clamp(sc.x + dx, 0, 100 - sc.w)
        y = clamp(sc.y + dy, 0, 100 - sc.h)
        break
      case 'nw': {
        const nx = clamp(sc.x + dx, 0, sc.x + sc.w - MIN_PCT)
        const ny = clamp(sc.y + dy, 0, sc.y + sc.h - MIN_PCT)
        w = sc.w - (nx - sc.x); h = sc.h - (ny - sc.y)
        x = nx; y = ny
        break
      }
      case 'ne': {
        const ny = clamp(sc.y + dy, 0, sc.y + sc.h - MIN_PCT)
        w = clamp(sc.w + dx, MIN_PCT, 100 - sc.x)
        h = sc.h - (ny - sc.y); y = ny
        break
      }
      case 'se':
        w = clamp(sc.w + dx, MIN_PCT, 100 - sc.x)
        h = clamp(sc.h + dy, MIN_PCT, 100 - sc.y)
        break
      case 'sw': {
        const nx = clamp(sc.x + dx, 0, sc.x + sc.w - MIN_PCT)
        w = sc.w - (nx - sc.x); h = clamp(sc.h + dy, MIN_PCT, 100 - sc.y)
        x = nx
        break
      }
    }

    const next = { x, y, w, h }
    cropRef.current = next
    setCrop(next)
  }

  function handlePointerUp() {
    dragRef.current = null
  }

  async function handleConfirm() {
    const bm = bitmapRef.current!
    setProcessing(true)
    try {
      const r = rotationRef.current
      const is90 = r === 90 || r === 270
      const nw = bm.width
      const nh = bm.height
      const rw = is90 ? nh : nw
      const rh = is90 ? nw : nh

      const full = document.createElement('canvas')
      full.width = rw
      full.height = rh
      const ctx = full.getContext('2d')!
      ctx.save()
      ctx.translate(rw / 2, rh / 2)
      ctx.rotate((r * Math.PI) / 180)
      ctx.drawImage(bm, -nw / 2, -nh / 2, nw, nh)
      ctx.restore()

      const c = cropRef.current
      const cropX = Math.round((c.x / 100) * rw)
      const cropY = Math.round((c.y / 100) * rh)
      const cropW = Math.max(1, Math.round((c.w / 100) * rw))
      const cropH = Math.max(1, Math.round((c.h / 100) * rh))

      const out = document.createElement('canvas')
      out.width = cropW
      out.height = cropH
      out.getContext('2d')!.drawImage(full, cropX, cropY, cropW, cropH, 0, 0, cropW, cropH)

      out.toBlob(
        (blob) => {
          if (blob) onConfirm(new File([blob], 'photo.jpg', { type: 'image/jpeg' }))
        },
        'image/jpeg',
        0.92,
      )
    } finally {
      setProcessing(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/95 flex flex-col select-none">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 flex-shrink-0">
        <button
          onClick={onCancel}
          className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm text-on-surface-dim hover:text-on-surface transition-fluid"
        >
          <X className="h-4 w-4" />
          Cancel
        </button>
        <span className="text-xs font-semibold text-on-surface-dim uppercase tracking-widest">Edit Photo</span>
        <button
          onClick={handleConfirm}
          disabled={!ready || processing}
          className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-sm font-semibold text-primary hover:bg-primary/10 disabled:opacity-40 transition-fluid"
        >
          {processing ? 'Processing…' : 'Use Photo'}
          <Check className="h-4 w-4" />
        </button>
      </div>

      {/* Canvas */}
      <div
        ref={containerRef}
        className="flex-1 flex items-center justify-center min-h-0 px-4 py-2"
      >
        {ready && (
          <canvas
            ref={canvasRef}
            className="block touch-none cursor-crosshair"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={handlePointerUp}
          />
        )}
        {!ready && (
          <div className="text-on-surface-faint text-sm">Loading…</div>
        )}
      </div>

      {/* Rotation controls */}
      <div className="flex items-center justify-center gap-3 px-4 py-3 flex-shrink-0">
        <button
          onClick={rotateLeft}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium bg-surface-container text-on-surface-dim hover:bg-surface-container-high transition-fluid"
        >
          <RotateCcw className="h-4 w-4" />
          Rotate Left
        </button>
        <button
          onClick={rotateRight}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium bg-surface-container text-on-surface-dim hover:bg-surface-container-high transition-fluid"
        >
          <RotateCw className="h-4 w-4" />
          Rotate Right
        </button>
      </div>
    </div>
  )
}

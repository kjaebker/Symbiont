import { useState, useRef, useEffect, useCallback } from 'react'
import { RotateCcw, RotateCw, X, Check, RefreshCcw } from 'lucide-react'
import { cn } from '@/lib/utils'

interface CropRect { x: number; y: number; w: number; h: number }
type DragHandle = 'move' | 'nw' | 'ne' | 'sw' | 'se' | 'n' | 's' | 'w' | 'e'

interface DragState {
  handle: DragHandle
  startX: number; startY: number
  origCrop: CropRect
}

interface AspectPreset { label: string; ratio: number | null }
const PRESETS: AspectPreset[] = [
  { label: 'Free',  ratio: null       },
  { label: '16:9',  ratio: 16 / 9     },
  { label: '4:3',   ratio: 4 / 3      },
  { label: '1:1',   ratio: 1          },
  { label: '9:16',  ratio: 9 / 16     },
]

interface Props {
  imageSrc: string       // preferred source (original backup when available)
  fallbackSrc?: string   // fallback if imageSrc 404s (the current processed image)
  onSave: (file: File) => Promise<void> | void
  onClose: () => void
}

const MIN_CROP_PX = 40

function drawRotated(
  ctx: CanvasRenderingContext2D,
  img: HTMLImageElement,
  rotation: number,
  dw: number,
  dh: number,
) {
  ctx.save()
  switch (rotation) {
    case 0:   ctx.drawImage(img, 0, 0, dw, dh); break
    case 90:  ctx.translate(dw, 0); ctx.rotate(Math.PI / 2); ctx.drawImage(img, 0, 0, dh, dw); break
    case 180: ctx.translate(dw, dh); ctx.rotate(Math.PI); ctx.drawImage(img, 0, 0, dw, dh); break
    case 270: ctx.translate(0, dh); ctx.rotate(-Math.PI / 2); ctx.drawImage(img, 0, 0, dh, dw); break
  }
  ctx.restore()
}

function fitToRatio(crop: CropRect, ratio: number): CropRect {
  const { x, y, w, h } = crop
  let nw = w, nh = h
  if (w / h > ratio) { nw = h * ratio } else { nh = w / ratio }
  return { x: x + (w - nw) / 2, y: y + (h - nh) / 2, w: nw, h: nh }
}

function constrainToRatio(
  handle: DragHandle,
  orig: CropRect,
  free: CropRect,
  ratio: number,
): CropRect {
  let { x, y, w, h } = free
  switch (handle) {
    case 'se': h = w / ratio; x = orig.x; y = orig.y; break
    case 'sw': { const r = orig.x + orig.w; h = w / ratio; x = r - w; y = orig.y; break }
    case 'ne': { const b = orig.y + orig.h; h = w / ratio; x = orig.x; y = b - h; break }
    case 'nw': { const r = orig.x + orig.w; const b = orig.y + orig.h; h = w / ratio; x = r - w; y = b - h; break }
    case 'n': case 's': { const cx = orig.x + orig.w / 2; w = h * ratio; x = cx - w / 2; break }
    case 'w': case 'e': { const cy = orig.y + orig.h / 2; h = w / ratio; y = cy - h / 2; break }
  }
  return { x, y, w, h }
}

export function ImageEditModal({ imageSrc, fallbackSrc, onSave, onClose }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const wrapRef = useRef<HTMLDivElement>(null)
  const imgRef = useRef<HTMLImageElement | null>(null)
  const dragRef = useRef<DragState | null>(null)

  const [rotation, setRotation] = useState(0)
  const [displayW, setDisplayW] = useState(0)
  const [displayH, setDisplayH] = useState(0)
  const [crop, setCrop] = useState<CropRect>({ x: 0, y: 0, w: 0, h: 0 })
  const [aspectRatio, setAspectRatio] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [imageLoaded, setImageLoaded] = useState(false)
  const [loadError, setLoadError] = useState(false)

  // Load image, fall back to fallbackSrc if the primary 404s
  useEffect(() => {
    setImageLoaded(false)
    setLoadError(false)

    const load = (src: string, fallback?: string) => {
      const img = new Image()
      img.onload = () => { imgRef.current = img; setImageLoaded(true) }
      img.onerror = () => {
        if (fallback && fallback !== src) { load(fallback) }
        else { setLoadError(true) }
      }
      img.src = src
    }
    load(imageSrc, fallbackSrc)
  }, [imageSrc, fallbackSrc])

  const computeDisplaySize = useCallback((rot: number): { w: number; h: number } => {
    const img = imgRef.current
    const wrap = wrapRef.current
    if (!img || !wrap) return { w: 0, h: 0 }
    const maxW = wrap.clientWidth - 32
    const maxH = Math.min(window.innerHeight * 0.5, 480)
    const sideways = rot === 90 || rot === 270
    const natW = sideways ? img.naturalHeight : img.naturalWidth
    const natH = sideways ? img.naturalWidth : img.naturalHeight
    const scale = Math.min(maxW / natW, maxH / natH, 1)
    return { w: Math.round(natW * scale), h: Math.round(natH * scale) }
  }, [])

  useEffect(() => {
    if (!imageLoaded) return
    const { w, h } = computeDisplaySize(rotation)
    setDisplayW(w)
    setDisplayH(h)
    const newCrop = { x: 0, y: 0, w, h }
    setCrop(aspectRatio ? fitToRatio(newCrop, aspectRatio) : newCrop)
  }, [imageLoaded, rotation, computeDisplaySize]) // intentionally exclude aspectRatio

  useEffect(() => {
    const canvas = canvasRef.current
    const img = imgRef.current
    if (!canvas || !img || displayW === 0) return
    canvas.width = displayW
    canvas.height = displayH
    drawRotated(canvas.getContext('2d')!, img, rotation, displayW, displayH)
  }, [displayW, displayH, rotation])

  function handlePresetChange(ratio: number | null) {
    setAspectRatio(ratio)
    if (ratio) setCrop(prev => fitToRatio(prev, ratio))
  }

  function rotate(dir: 'cw' | 'ccw') {
    setRotation(r => (r + (dir === 'cw' ? 90 : 270)) % 360)
  }

  function resetCrop() {
    const full = { x: 0, y: 0, w: displayW, h: displayH }
    setCrop(aspectRatio ? fitToRatio(full, aspectRatio) : full)
  }

  function onPointerDown(e: React.PointerEvent<HTMLDivElement>, handle: DragHandle) {
    e.stopPropagation()
    ;(e.currentTarget as HTMLDivElement).setPointerCapture(e.pointerId)
    dragRef.current = { handle, startX: e.clientX, startY: e.clientY, origCrop: { ...crop } }
  }

  function onPointerMove(e: React.PointerEvent<HTMLDivElement>) {
    const d = dragRef.current
    if (!d) return
    const dx = e.clientX - d.startX
    const dy = e.clientY - d.startY
    const { x, y, w, h } = d.origCrop
    let nx = x, ny = y, nw = w, nh = h

    switch (d.handle) {
      case 'move': nx = Math.max(0, Math.min(displayW - w, x + dx)); ny = Math.max(0, Math.min(displayH - h, y + dy)); break
      case 'nw':   nx = Math.max(0, Math.min(x + w - MIN_CROP_PX, x + dx)); ny = Math.max(0, Math.min(y + h - MIN_CROP_PX, y + dy)); nw = x + w - nx; nh = y + h - ny; break
      case 'ne':   ny = Math.max(0, Math.min(y + h - MIN_CROP_PX, y + dy)); nw = Math.max(MIN_CROP_PX, Math.min(displayW - x, w + dx)); nh = y + h - ny; break
      case 'sw':   nx = Math.max(0, Math.min(x + w - MIN_CROP_PX, x + dx)); nw = x + w - nx; nh = Math.max(MIN_CROP_PX, Math.min(displayH - y, h + dy)); break
      case 'se':   nw = Math.max(MIN_CROP_PX, Math.min(displayW - x, w + dx)); nh = Math.max(MIN_CROP_PX, Math.min(displayH - y, h + dy)); break
      case 'n':    ny = Math.max(0, Math.min(y + h - MIN_CROP_PX, y + dy)); nh = y + h - ny; break
      case 's':    nh = Math.max(MIN_CROP_PX, Math.min(displayH - y, h + dy)); break
      case 'w':    nx = Math.max(0, Math.min(x + w - MIN_CROP_PX, x + dx)); nw = x + w - nx; break
      case 'e':    nw = Math.max(MIN_CROP_PX, Math.min(displayW - x, w + dx)); break
    }

    let next: CropRect = { x: nx, y: ny, w: nw, h: nh }
    if (aspectRatio && d.handle !== 'move') {
      next = constrainToRatio(d.handle, d.origCrop, next, aspectRatio)
      // clamp to display bounds after ratio constraint
      next.x = Math.max(0, next.x)
      next.y = Math.max(0, next.y)
      if (next.x + next.w > displayW) next.w = displayW - next.x
      if (next.y + next.h > displayH) next.h = displayH - next.y
      next.w = Math.max(MIN_CROP_PX, next.w)
      next.h = Math.max(MIN_CROP_PX, next.h)
    }

    setCrop(next)
  }

  function onPointerUp() { dragRef.current = null }

  async function handleSave() {
    const img = imgRef.current
    if (!img || displayW === 0) return
    setSaving(true)
    try {
      const sideways = rotation === 90 || rotation === 270
      const fullW = sideways ? img.naturalHeight : img.naturalWidth
      const fullH = sideways ? img.naturalWidth : img.naturalHeight

      const rotCanvas = document.createElement('canvas')
      rotCanvas.width = fullW; rotCanvas.height = fullH
      drawRotated(rotCanvas.getContext('2d')!, img, rotation, fullW, fullH)

      const sx = fullW / displayW, sy = fullH / displayH
      const outCanvas = document.createElement('canvas')
      outCanvas.width = Math.round(crop.w * sx)
      outCanvas.height = Math.round(crop.h * sy)
      outCanvas.getContext('2d')!.drawImage(
        rotCanvas,
        Math.round(crop.x * sx), Math.round(crop.y * sy),
        outCanvas.width, outCanvas.height,
        0, 0, outCanvas.width, outCanvas.height,
      )

      const blob = await new Promise<Blob>((resolve, reject) => {
        outCanvas.toBlob(b => (b ? resolve(b) : reject(new Error('canvas export failed'))), 'image/jpeg', 0.9)
      })
      await onSave(new File([blob], 'edited.jpg', { type: 'image/jpeg' }))
    } finally {
      setSaving(false)
    }
  }

  const isCropFull = crop.x === 0 && crop.y === 0 && crop.w === displayW && crop.h === displayH

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: 'rgba(7, 13, 31, 0.92)', backdropFilter: 'blur(8px)' }}
    >
      <div className="w-full max-w-2xl mx-4 flex flex-col gap-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-on-surface">Edit Photo</span>
          <button onClick={onClose} className="p-1.5 rounded-lg text-on-surface-faint hover:text-on-surface transition-fluid">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Canvas + crop overlay */}
        <div
          ref={wrapRef}
          className="w-full bg-surface-container rounded-2xl flex items-center justify-center p-4 min-h-40"
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerLeave={onPointerUp}
        >
          {loadError ? (
            <p className="text-on-surface-faint text-sm">Failed to load image.</p>
          ) : !imageLoaded ? (
            <div className="w-full h-48 animate-pulse bg-surface-container-high rounded-xl" />
          ) : (
            <div className="relative select-none" style={{ width: displayW, height: displayH }}>
              <canvas ref={canvasRef} style={{ width: displayW, height: displayH, display: 'block' }} />

              {/* Dark masks outside crop */}
              <div className="absolute pointer-events-none bg-black/60" style={{ top: 0, left: 0, right: 0, height: crop.y }} />
              <div className="absolute pointer-events-none bg-black/60" style={{ top: crop.y + crop.h, left: 0, right: 0, bottom: 0 }} />
              <div className="absolute pointer-events-none bg-black/60" style={{ top: crop.y, left: 0, width: crop.x, height: crop.h }} />
              <div className="absolute pointer-events-none bg-black/60" style={{ top: crop.y, left: crop.x + crop.w, right: 0, height: crop.h }} />

              {/* Crop border */}
              <div className="absolute pointer-events-none" style={{ top: crop.y, left: crop.x, width: crop.w, height: crop.h, boxShadow: 'inset 0 0 0 1.5px rgba(58,223,250,0.7)' }} />

              {/* Rule-of-thirds */}
              {!isCropFull && (<>
                <div className="absolute pointer-events-none bg-white/10" style={{ top: crop.y + crop.h / 3, left: crop.x, width: crop.w, height: 1 }} />
                <div className="absolute pointer-events-none bg-white/10" style={{ top: crop.y + (crop.h * 2) / 3, left: crop.x, width: crop.w, height: 1 }} />
                <div className="absolute pointer-events-none bg-white/10" style={{ top: crop.y, left: crop.x + crop.w / 3, width: 1, height: crop.h }} />
                <div className="absolute pointer-events-none bg-white/10" style={{ top: crop.y, left: crop.x + (crop.w * 2) / 3, width: 1, height: crop.h }} />
              </>)}

              {/* Move zone */}
              <div className="absolute" style={{ top: crop.y + 18, left: crop.x + 18, width: Math.max(0, crop.w - 36), height: Math.max(0, crop.h - 36), cursor: 'move' }} onPointerDown={e => onPointerDown(e, 'move')} />

              {/* Corner handles */}
              {([['nw', crop.x - 5, crop.y - 5, 'nw-resize'], ['ne', crop.x + crop.w - 7, crop.y - 5, 'ne-resize'], ['sw', crop.x - 5, crop.y + crop.h - 7, 'sw-resize'], ['se', crop.x + crop.w - 7, crop.y + crop.h - 7, 'se-resize']] as const).map(([h, lx, ly, cur]) => (
                <div key={h} className="absolute w-3 h-3 rounded-sm bg-primary" style={{ left: lx, top: ly, cursor: cur }} onPointerDown={e => onPointerDown(e, h)} />
              ))}

              {/* Edge handles */}
              {([['n', crop.x + crop.w / 2 - 6, crop.y - 4, 'n-resize', 12, 8], ['s', crop.x + crop.w / 2 - 6, crop.y + crop.h - 4, 's-resize', 12, 8], ['w', crop.x - 4, crop.y + crop.h / 2 - 6, 'w-resize', 8, 12], ['e', crop.x + crop.w - 4, crop.y + crop.h / 2 - 6, 'e-resize', 8, 12]] as const).map(([h, lx, ly, cur, bw, bh]) => (
                <div key={h} className="absolute rounded-sm bg-primary" style={{ left: lx, top: ly, width: bw, height: bh, cursor: cur }} onPointerDown={e => onPointerDown(e, h)} />
              ))}
            </div>
          )}
        </div>

        {/* Aspect ratio presets */}
        <div className="flex items-center gap-2">
          {PRESETS.map(p => (
            <button
              key={p.label}
              onClick={() => handlePresetChange(p.ratio)}
              className={cn(
                'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
                aspectRatio === p.ratio
                  ? 'bg-primary/20 text-primary'
                  : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high hover:text-on-surface',
              )}
            >
              {p.label}
            </button>
          ))}
        </div>

        {/* Bottom controls */}
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <button onClick={() => rotate('ccw')} title="Rotate counter-clockwise" className="p-2 rounded-xl bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid">
              <RotateCcw className="h-4 w-4" />
            </button>
            <button onClick={() => rotate('cw')} title="Rotate clockwise" className="p-2 rounded-xl bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid">
              <RotateCw className="h-4 w-4" />
            </button>
            {!isCropFull && (
              <button onClick={resetCrop} className="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-surface-container text-on-surface-dim text-xs hover:text-on-surface hover:bg-surface-container-high transition-fluid">
                <RefreshCcw className="h-3.5 w-3.5" />
                Reset crop
              </button>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button onClick={onClose} disabled={saving} className="px-4 py-2 rounded-xl text-sm font-medium bg-surface-container text-on-surface-dim hover:bg-surface-container-high disabled:opacity-40 transition-fluid">
              Cancel
            </button>
            <button onClick={handleSave} disabled={saving || !imageLoaded} className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-primary/20 text-primary hover:bg-primary/30 disabled:opacity-40 transition-fluid">
              <Check className="h-4 w-4" />
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

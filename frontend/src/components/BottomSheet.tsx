import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface BottomSheetProps {
  open: boolean
  onClose: () => void
  title: string
  children: React.ReactNode
}

export function BottomSheet({ open, onClose, title, children }: BottomSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null)
  const swipeStartY = useRef<number | null>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  function handleDragStart(e: React.TouchEvent) {
    swipeStartY.current = e.touches[0].clientY
    if (sheetRef.current) sheetRef.current.style.transition = 'none'
  }

  function handleDragMove(e: React.TouchEvent) {
    if (swipeStartY.current === null) return
    const dy = e.touches[0].clientY - swipeStartY.current
    if (dy > 0 && sheetRef.current) {
      sheetRef.current.style.transform = `translateY(${dy}px)`
    }
  }

  function handleDragEnd(e: React.TouchEvent) {
    if (swipeStartY.current === null) return
    const dy = e.changedTouches[0].clientY - swipeStartY.current
    swipeStartY.current = null
    if (sheetRef.current) {
      sheetRef.current.style.transition = ''
      sheetRef.current.style.transform = ''
    }
    if (dy > 80) onClose()
  }

  return createPortal(
    <>
      {/* Backdrop */}
      <div
        className={cn(
          'md:hidden fixed inset-0 z-50 bg-black/50 transition-opacity duration-300',
          open ? 'opacity-100' : 'opacity-0 pointer-events-none',
        )}
        onClick={onClose}
        aria-hidden="true"
      />
      {/* Sheet */}
      <div
        ref={sheetRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={cn(
          'md:hidden fixed left-0 right-0 z-50 flex flex-col rounded-t-3xl bg-surface-container transition-transform duration-300 ease-out',
          open ? 'translate-y-0' : 'translate-y-full',
        )}
        style={{ bottom: 0, height: 'calc(85vh - 50px)', overflow: 'hidden' }}
      >
        {/* Drag handle — swipe target */}
        <div
          className="flex justify-center pt-4 pb-3 shrink-0 touch-none select-none"
          onTouchStart={handleDragStart}
          onTouchMove={handleDragMove}
          onTouchEnd={handleDragEnd}
        >
          <div className="w-9 h-1 rounded-full bg-outline-variant/40" />
        </div>
        {/* Header — also a swipe target */}
        <div
          className="flex items-center justify-between px-5 py-3 shrink-0 select-none"
          onTouchStart={handleDragStart}
          onTouchMove={handleDragMove}
          onTouchEnd={handleDragEnd}
        >
          <span className="text-sm font-semibold text-on-surface uppercase tracking-widest">
            {title}
          </span>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid"
            aria-label="Close"
          >
            <X size={16} />
          </button>
        </div>
        {/* Content */}
        <div className="px-5 pb-8 overflow-y-auto flex-1 min-h-0">
          {children}
        </div>
      </div>
    </>,
    document.body,
  )
}

import { useEffect } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface BottomSheetProps {
  open: boolean
  onClose: () => void
  title: string
  children: React.ReactNode
}

export function BottomSheet({ open, onClose, title, children }: BottomSheetProps) {
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  return (
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
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={cn(
          'md:hidden fixed left-0 right-0 z-50 flex flex-col rounded-t-3xl bg-surface-container transition-transform duration-300 ease-out',
          open ? 'translate-y-0' : 'translate-y-full',
        )}
        style={{ bottom: 0, maxHeight: '80vh' }}
      >
        {/* Drag handle */}
        <div className="flex justify-center pt-3 pb-1 shrink-0">
          <div className="w-9 h-1 rounded-full bg-outline-variant/40" />
        </div>
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 shrink-0">
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
    </>
  )
}

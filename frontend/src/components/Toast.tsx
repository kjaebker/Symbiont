import { useState, useEffect, useCallback, createContext, useContext, type ReactNode } from 'react'
import { X, AlertTriangle, CheckCircle, Info, Bell } from 'lucide-react'
import { cn } from '@/lib/utils'

type ToastType = 'info' | 'success' | 'warning' | 'error' | 'alert'

interface Toast {
  id: number
  type: ToastType
  message: string
}

interface ToastContextValue {
  addToast: (type: ToastType, message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = useCallback((type: ToastType, message: string) => {
    const id = nextId++
    setToasts((prev) => [...prev, { id, type, message }])
  }, [])

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      <div className="fixed top-4 right-4 z-[100] flex flex-col gap-2 max-w-sm">
        {toasts.map((toast) => (
          <ToastItem key={toast.id} toast={toast} onDismiss={removeToast} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

const typeStyles: Record<ToastType, string> = {
  info: 'text-primary',
  success: 'text-secondary',
  warning: 'text-amber-400',
  error: 'text-tertiary',
  alert: 'text-tertiary',
}

const typeIcons: Record<ToastType, typeof Info> = {
  info: Info,
  success: CheckCircle,
  warning: AlertTriangle,
  error: AlertTriangle,
  alert: Bell,
}

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: (id: number) => void }) {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
    const timer = setTimeout(() => {
      setVisible(false)
      setTimeout(() => onDismiss(toast.id), 300)
    }, 5000)
    return () => clearTimeout(timer)
  }, [toast.id, onDismiss])

  const Icon = typeIcons[toast.type]

  return (
    <div
      className={cn(
        'glass rounded-xl p-3 flex items-start gap-3 shadow-abyss transition-all duration-300',
        visible ? 'opacity-100 translate-x-0' : 'opacity-0 translate-x-4',
      )}
    >
      <Icon size={16} className={cn('shrink-0 mt-0.5', typeStyles[toast.type])} />
      <p className="text-sm text-on-surface flex-1">{toast.message}</p>
      <button
        onClick={() => {
          setVisible(false)
          setTimeout(() => onDismiss(toast.id), 300)
        }}
        className="text-on-surface-faint hover:text-on-surface-dim shrink-0"
      >
        <X size={14} />
      </button>
    </div>
  )
}

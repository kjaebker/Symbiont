import { NavLink, Outlet, useLocation } from 'react-router-dom'
import {
  LayoutDashboard,
  BookOpen,
  FlaskConical,
  Fish,
  Bell,
  Settings,
  LayoutGrid,
  X,
  Wrench,
} from 'lucide-react'
import { useState, useEffect, useCallback } from 'react'
import { useSSE } from '@/hooks/useSSE'
import { useSystemStatus } from '@/hooks/useSystem'
import { cn, relativeTime } from '@/lib/utils'
import { getBubblesEnabled } from '@/api/client'
import { Bubbles } from '@/components/Bubbles'

const primaryNavItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/livestock', icon: Fish, label: 'Livestock' },
  { to: '/chemistry', icon: FlaskConical, label: 'Chemistry' },
  { to: '/maintenance', icon: Wrench, label: 'Maintenance' },
]

const overflowNavItems = [
  { to: '/history', icon: BookOpen, label: 'Log' },
  { to: '/alerts', icon: Bell, label: 'Alerts' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

const allNavItems = [...primaryNavItems, ...overflowNavItems]

function StatusBadge() {
  const { data } = useSystemStatus()
  const pollOk = data?.poller.poll_ok ?? false
  const lastPoll = data?.poller.last_poll_ts

  return (
    <div className="flex items-center gap-2 text-sm text-on-surface-dim">
      <span
        className={cn(
          'h-2 w-2 rounded-full',
          pollOk
            ? 'bg-secondary animate-bio-pulse'
            : 'bg-tertiary',
        )}
      />
      <span className="hidden md:inline">
        {lastPoll ? relativeTime(lastPoll) : 'Connecting...'}
      </span>
    </div>
  )
}


function MobileMoreSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const location = useLocation()

  // Close sheet when route changes
  useEffect(() => {
    onClose()
  }, [location.pathname, onClose])

  return (
    <>
      {/* Backdrop */}
      <div
        className={cn(
          'md:hidden fixed inset-0 z-40 bg-surface/60 backdrop-blur-sm transition-fluid',
          open ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none',
        )}
        onClick={onClose}
      />

      {/* Sheet — anchored at bottom-0, pb clears the nav bar, z-index below nav */}
      <div
        className={cn(
          'md:hidden fixed bottom-0 inset-x-0 pb-[4.5rem] transition-fluid',
          open ? 'translate-y-0' : 'translate-y-full',
        )}
        style={{ zIndex: 49 }}
      >
        <div className="mx-3 mb-2 rounded-2xl bg-surface-container-low/95 backdrop-blur-xl overflow-hidden"
          style={{ boxShadow: '0 -4px 40px rgba(58, 223, 250, 0.08)' }}>
          {/* Sheet header */}
          <div className="flex items-center justify-end px-4 pt-3 pb-1">
            <button
              onClick={onClose}
              className="p-1.5 rounded-full text-on-surface-dim hover:text-on-surface hover:bg-surface-container/60 transition-fluid"
            >
              <X size={18} />
            </button>
          </div>

          {/* Overflow nav items */}
          <div className="px-3 pb-4 flex flex-col gap-1">
            {overflowNavItems.map(({ to, icon: Icon, label }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-4 px-4 py-3 rounded-xl text-sm font-medium transition-fluid',
                    isActive
                      ? 'bg-surface-container-high text-primary shadow-glow-primary'
                      : 'text-on-surface-dim hover:text-on-surface hover:bg-surface-container/60',
                  )
                }
              >
                <Icon size={20} />
                {label}
              </NavLink>
            ))}
          </div>
        </div>
      </div>
    </>
  )
}

export default function Layout() {
  useSSE()
  const location = useLocation()
  const [moreOpen, setMoreOpen] = useState(false)
  const closeMore = useCallback(() => setMoreOpen(false), [])
  const bubblesEnabled = getBubblesEnabled()

  const activeRouteIsOverflow = overflowNavItems.some(
    (item) => item.to === location.pathname,
  )

  return (
    <div className="flex h-screen overflow-hidden">
      {bubblesEnabled && <Bubbles />}

      {/* Desktop sidebar */}
      <nav className="hidden md:flex flex-col w-56 bg-surface-container-low p-4 gap-1 shrink-0">
        <div className="flex items-center gap-3 px-2 py-3 mb-4">
          <div className="w-12 h-12 rounded-xl overflow-hidden shrink-0">
            <img src="/icon-192.png" alt="" className="w-full h-full object-cover scale-150" />
          </div>
          <span className="text-lg font-bold text-on-surface tracking-tight">Symbiont</span>
        </div>

        {allNavItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-fluid',
                isActive
                  ? 'bg-surface-container-high text-primary shadow-glow-primary'
                  : 'text-on-surface-dim hover:text-on-surface hover:bg-surface-container/60',
              )
            }
          >
            <Icon size={18} />
            {label}
          </NavLink>
        ))}

        <div className="mt-auto px-3 py-3">
          <StatusBadge />
        </div>
      </nav>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto pb-20 md:pb-0">
        <Outlet />
      </main>

      {/* Mobile bottom nav — z-50 stays above the sheet (z-49) */}
      <nav className="md:hidden fixed bottom-0 inset-x-0 bg-surface-container-low/90 backdrop-blur-lg z-50">
        <div className="flex items-center justify-around py-2">
          {primaryNavItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                cn(
                  'flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl text-xs transition-fluid',
                  isActive ? 'text-primary' : 'text-on-surface-dim',
                )
              }
            >
              <Icon size={20} />
              <span>{label}</span>
            </NavLink>
          ))}

          {/* More button */}
          <button
            onClick={() => setMoreOpen((v) => !v)}
            className={cn(
              'flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl text-xs transition-fluid',
              moreOpen || activeRouteIsOverflow ? 'text-primary' : 'text-on-surface-dim',
            )}
          >
            <LayoutGrid size={20} />
            <span>More</span>
          </button>
        </div>
      </nav>

      {/* Mobile overflow sheet */}
      <MobileMoreSheet open={moreOpen} onClose={closeMore} />
    </div>
  )
}

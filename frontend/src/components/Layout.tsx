import { NavLink, Outlet } from 'react-router-dom'
import {
  LayoutDashboard,
  Clock,
  Power,
  Bell,
  Settings,
} from 'lucide-react'
import { useSSE } from '@/hooks/useSSE'
import { useSystemStatus } from '@/hooks/useSystem'
import { cn, relativeTime } from '@/lib/utils'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/history', icon: Clock, label: 'History' },
  { to: '/outlets', icon: Power, label: 'Outlets' },
  { to: '/alerts', icon: Bell, label: 'Alerts' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

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

export default function Layout() {
  useSSE()

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Desktop sidebar */}
      <nav className="hidden md:flex flex-col w-56 bg-surface-container-low p-4 gap-1 shrink-0">
        <div className="px-2 py-3 mb-4">
          <img src="/logo.png" alt="Symbiont" className="w-full object-contain" style={{ maxHeight: '72px' }} />
        </div>

        {navItems.map(({ to, icon: Icon, label }) => (
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

      {/* Mobile bottom nav */}
      <nav className="md:hidden fixed bottom-0 inset-x-0 bg-surface-container-low/90 backdrop-blur-lg z-50">
        <div className="flex items-center justify-around py-2">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                cn(
                  'flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl text-xs transition-fluid',
                  isActive
                    ? 'text-primary'
                    : 'text-on-surface-dim',
                )
              }
            >
              <Icon size={20} />
              <span>{label}</span>
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  )
}

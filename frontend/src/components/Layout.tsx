import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  BookOpen,
  FlaskConical,
  Fish,
  Bell,
  Settings,
  Wrench,
} from 'lucide-react'
import { useState, useEffect, useCallback, useRef } from 'react'
import { useSSE } from '@/hooks/useSSE'
import { useSystemStatus } from '@/hooks/useSystem'
import { cn, relativeTime } from '@/lib/utils'
import { getBubblesEnabled } from '@/api/client'
import { Bubbles } from '@/components/Bubbles'
import { Caustic } from '@/components/Caustic'
import { SidebarSilhouette } from '@/components/SidebarSilhouette'
import { routeThemes, defaultRouteTheme, PageThemeContext } from '@/lib/pageTheme'
import type { PageTheme } from '@/lib/pageTheme'

function hexToGlow(hex: string, alpha = 0.3): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

const allNavItems = [
  { to: '/',            icon: LayoutDashboard, label: 'Dashboard'   },
  { to: '/livestock',   icon: Fish,            label: 'Livestock'   },
  { to: '/chemistry',   icon: FlaskConical,    label: 'Chemistry'   },
  { to: '/maintenance', icon: Wrench,          label: 'Maintenance' },
  { to: '/history',     icon: BookOpen,        label: 'Log'         },
  { to: '/alerts',      icon: Bell,            label: 'Alerts'      },
  { to: '/settings',    icon: Settings,        label: 'Settings'    },
]

const mobileNavItems = allNavItems.map(item => {
  const theme = routeThemes[item.to] ?? defaultRouteTheme
  return { ...item, color: theme.accent, glow: hexToGlow(theme.accent) }
})

function getActiveNavItem(pathname: string) {
  return (
    mobileNavItems.find(item => item.to === pathname) ??
    mobileNavItems.find(item => item.to !== '/' && pathname.startsWith(item.to)) ??
    mobileNavItems[0]
  )
}

function StatusBadge() {
  const { data } = useSystemStatus()
  const pollOk = data?.poller.poll_ok ?? false
  const lastPoll = data?.poller.last_poll_ts

  return (
    <div className="flex items-center gap-2 text-sm text-on-surface-dim">
      <span
        className={cn(
          'h-2 w-2 rounded-full',
          pollOk ? 'bg-secondary animate-bio-pulse' : 'bg-tertiary',
        )}
      />
      <span className="hidden md:inline">
        {lastPoll ? relativeTime(lastPoll) : 'Connecting...'}
      </span>
    </div>
  )
}

interface MobileDrawerProps {
  open: boolean
  onClose: () => void
  activeItem: typeof mobileNavItems[0]
}

function MobileDrawer({ open, onClose, activeItem }: MobileDrawerProps) {
  const navigate = useNavigate()
  const { data: status } = useSystemStatus()
  const [hovered, setHovered] = useState<string | null>(null)

  const handleNav = (to: string) => {
    navigate(to)
    onClose()
  }

  return (
    <div
      className="md:hidden absolute inset-y-0 left-0 flex flex-col overflow-hidden"
      style={{ width: 'min(318px, 85vw)', zIndex: 1, background: '#090f22' }}
      role="navigation"
      aria-label="Main navigation"
    >
      {/* Ambient glow — shifts color with active page */}
      <div
        aria-hidden="true"
        style={{
          position: 'absolute',
          top: '25%',
          left: '-30%',
          width: 280,
          height: 280,
          borderRadius: '50%',
          background: `radial-gradient(circle, ${activeItem.color}14 0%, transparent 70%)`,
          pointerEvents: 'none',
          transition: 'background 500ms ease',
        }}
      />

      {/* Header — matches desktop sidebar */}
      <div style={{ padding: 'calc(env(safe-area-inset-top) + 58px) 22px 18px', position: 'relative' }}>
        <div className="flex items-center gap-3 px-0 py-0">
          <div className="w-12 h-12 rounded-xl overflow-hidden shrink-0">
            <img src="/icon-192.png" alt="" className="w-full h-full object-cover scale-150" />
          </div>
          <span className="text-lg font-bold text-on-surface tracking-tight">Symbiont</span>
        </div>
      </div>

      {/* Nav items — key on open to replay stagger animation each reveal */}
      <div
        key={open ? 'open' : 'closed'}
        style={{ flex: 1, overflowY: 'auto', padding: '4px 14px 8px' }}
      >
        {mobileNavItems.map((item, i) => {
          const isActive = activeItem.to === item.to
          const isHov = hovered === item.to && !isActive
          const Icon = item.icon

          return (
            <button
              key={item.to}
              onClick={() => handleNav(item.to)}
              onMouseEnter={() => setHovered(item.to)}
              onMouseLeave={() => setHovered(null)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 14,
                width: '100%',
                padding: '13px 14px',
                borderRadius: 14,
                border: 'none',
                cursor: 'pointer',
                marginBottom: 3,
                background: isActive
                  ? `linear-gradient(110deg, ${item.color}22, ${item.color}0d)`
                  : isHov
                  ? '#11192e'
                  : 'transparent',
                boxShadow: isActive
                  ? `inset 0 0 0 1px ${item.color}38, 0 4px 20px ${item.glow}`
                  : 'none',
                transition: 'all 180ms ease',
                position: 'relative',
                overflow: 'hidden',
                animation: `nav-item-in 240ms cubic-bezier(0.34,1.4,0.64,1) ${i * 35}ms both`,
              }}
            >
              {/* Left accent bar on active item */}
              {isActive && (
                <div
                  style={{
                    position: 'absolute',
                    left: 0,
                    top: '18%',
                    bottom: '18%',
                    width: 3,
                    borderRadius: 9999,
                    background: `linear-gradient(180deg, ${item.color}, ${item.color}80)`,
                    boxShadow: `0 0 8px ${item.glow}`,
                  }}
                />
              )}

              {/* Icon container */}
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: 12,
                  flexShrink: 0,
                  background: isActive ? `${item.color}22` : '#171f36',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  boxShadow: isActive ? `0 0 14px ${item.glow}` : 'none',
                  transition: 'all 180ms ease',
                }}
              >
                <Icon
                  size={19}
                  strokeWidth={1.75}
                  color={isActive ? item.color : isHov ? '#8a90a8' : '#5a6080'}
                />
              </div>

              {/* Label */}
              <span
                style={{
                  fontSize: 15,
                  fontWeight: isActive ? 700 : 500,
                  color: isActive ? item.color : isHov ? '#dfe4fe' : '#8a90a8',
                  letterSpacing: isActive ? '-0.02em' : '-0.01em',
                  flex: 1,
                  textAlign: 'left',
                  transition: 'all 180ms ease',
                }}
              >
                {item.label}
              </span>

              {/* Active pulse dot */}
              {isActive && (
                <div
                  className="animate-bio-pulse"
                  style={{
                    width: 7,
                    height: 7,
                    borderRadius: '50%',
                    background: item.color,
                    boxShadow: `0 0 10px ${item.glow}`,
                    flexShrink: 0,
                  }}
                />
              )}
            </button>
          )
        })}
      </div>

      {/* Footer status */}
      <div style={{ padding: '12px 20px 28px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '10px 14px',
            borderRadius: 10,
            background: '#11192e',
          }}
        >
          <div
            className="animate-bio-pulse"
            style={{ width: 6, height: 6, borderRadius: '50%', background: '#6dfe9c', flexShrink: 0 }}
          />
          <span style={{ fontSize: 11, color: '#8a90a8', flex: 1 }}>
            {status?.poller.poll_ok ? 'System healthy' : 'Connecting...'}
          </span>
          {status?.poller.last_poll_ts && (
            <span style={{ fontSize: 10, color: '#5a6080' }}>
              {relativeTime(status.poller.last_poll_ts)}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

interface MobileBottomBarProps {
  open: boolean
  onToggle: () => void
  activeItem: typeof mobileNavItems[0]
}

function MobileBottomBar({ open, onToggle, activeItem }: MobileBottomBarProps) {
  const navigate = useNavigate()
  const touchStartX = useRef<number | null>(null)

  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
  }

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartX.current === null) return
    const delta = e.changedTouches[0].clientX - touchStartX.current
    touchStartX.current = null
    if (Math.abs(delta) < 40) return

    const idx = mobileNavItems.findIndex(item => item.to === activeItem.to)
    const target = delta < 0 ? mobileNavItems[idx + 1] : mobileNavItems[idx - 1]
    if (target) navigate(target.to)
  }

  const Icon = activeItem.icon

  return (
    <div className="md:hidden shrink-0" style={{ position: 'relative', zIndex: 20 }} onTouchStart={handleTouchStart} onTouchEnd={handleTouchEnd}>
      {/* Visible bar — 66px, sits above the home indicator zone */}
      <div
        className="flex items-center"
        style={{
          height: 56,
          background: '#0c1326',
          borderTop: '1px solid rgba(65,71,91,0.15)',
          position: 'relative',
          zIndex: 1,
          paddingLeft: 16,
          paddingRight: 18,
          gap: 10,
        }}
      >
      {/* Hamburger button */}
      <button
        onClick={onToggle}
        aria-expanded={open}
        aria-label="Toggle navigation"
        style={{
          width: 44,
          height: 44,
          borderRadius: 12,
          border: 'none',
          cursor: 'pointer',
          background: open ? 'rgba(58,223,250,0.12)' : 'transparent',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 4.5,
          transition: 'all 200ms ease',
          flexShrink: 0,
        }}
      >
        <div style={{ width: 15, height: 1.5, borderRadius: 9, background: open ? '#3adffa' : '#8a90a8', transition: 'all 200ms ease' }} />
        <div style={{ width: 11, height: 1.5, borderRadius: 9, background: open ? '#3adffa' : '#5a6080', transition: 'all 200ms ease' }} />
        <div style={{ width: 15, height: 1.5, borderRadius: 9, background: open ? '#3adffa' : '#8a90a8', transition: 'all 200ms ease' }} />
      </button>

      {/* Vertical rule */}
      <div style={{ width: 1, height: 20, background: 'rgba(65,71,91,0.4)', flexShrink: 0 }} />

      {/* Active page indicator */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 7, flex: 1, minWidth: 0 }}>
        <Icon size={16} strokeWidth={1.75} color={activeItem.color} />
        <span
          style={{
            fontSize: 13,
            fontWeight: 700,
            color: activeItem.color,
            letterSpacing: '-0.01em',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {activeItem.label}
        </span>
      </div>

      {/* Quick-jump dots — one per page, tap to navigate, swipe bar to step through */}
      <div style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
        {mobileNavItems.map(item => {
          const isActive = item.to === activeItem.to
          return (
            <button
              key={item.to}
              onClick={() => navigate(item.to)}
              aria-label={item.label}
              style={{
                padding: '10px 4px',
                border: 'none',
                cursor: 'pointer',
                background: 'transparent',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <div style={{
                width: isActive ? 20 : 6,
                height: 6,
                borderRadius: 9999,
                background: isActive ? item.color : '#1c253e',
                boxShadow: isActive ? `0 0 8px ${item.glow}` : 'none',
                transition: 'all 280ms cubic-bezier(0.34,1.4,0.64,1)',
              }} />
            </button>
          )
        })}
      </div>
      </div>
      {/* Safe-area filler — 34px below bar matches home indicator zone */}
      <div style={{ height: 'env(safe-area-inset-bottom, 0px)', background: '#0c1326' }} />
    </div>
  )
}

export default function Layout() {
  useSSE()
  const location = useLocation()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const closeDrawer = useCallback(() => setDrawerOpen(false), [])
  const bubblesEnabled = getBubblesEnabled()

  const routeDefault = routeThemes[location.pathname] ?? defaultRouteTheme
  const [theme, setThemeState] = useState<Pick<PageTheme, 'accent' | 'sub'>>({
    accent: routeDefault.accent,
    sub: routeDefault.sub,
  })

  // Reset theme and close drawer on navigation
  useEffect(() => {
    const rd = routeThemes[location.pathname] ?? defaultRouteTheme
    setThemeState({ accent: rd.accent, sub: rd.sub })
    setDrawerOpen(false)
  }, [location.pathname])

  const ctx: PageTheme = {
    ...theme,
    page: routeDefault.page,
    setTheme: (t) => setThemeState((prev) => ({ ...prev, ...t })),
  }

  const activeNavItem = getActiveNavItem(location.pathname)

  const contentPanelStyle: React.CSSProperties = {
    transform: drawerOpen ? 'translateX(min(318px, 85vw))' : 'translateX(0)',
    borderRadius: drawerOpen ? '20px 0 0 20px' : '0',
    boxShadow: drawerOpen ? '-20px 0 60px rgba(0,0,0,0.6)' : 'none',
    transition: [
      'transform 320ms cubic-bezier(0.32,0.72,0,1)',
      'border-radius 320ms cubic-bezier(0.32,0.72,0,1)',
      'box-shadow 320ms cubic-bezier(0.32,0.72,0,1)',
    ].join(', '),
    zIndex: 10,
  }

  return (
    <PageThemeContext.Provider value={ctx}>
      <div className="fixed inset-0 flex overflow-hidden">

        {/* Desktop sidebar — unchanged */}
        <nav className="hidden md:flex flex-col w-56 bg-surface-container-low p-4 gap-1 shrink-0 relative overflow-hidden">
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
                  'flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-fluid relative z-10',
                  isActive
                    ? 'bg-surface-container-high'
                    : 'text-on-surface-dim hover:text-on-surface hover:bg-surface-container/60',
                )
              }
              style={({ isActive }) =>
                isActive ? { color: ctx.accent, boxShadow: `0 0 20px ${ctx.accent}22` } : {}
              }
            >
              <Icon size={18} />
              {label}
            </NavLink>
          ))}

          <div className="mt-auto px-3 py-3 relative z-10">
            <StatusBadge />
          </div>

          <SidebarSilhouette page={routeDefault.page} />
        </nav>

        {/* Right side: stacking context for mobile push-reveal */}
        <div className="flex-1 relative overflow-hidden flex flex-col">

          {/* Mobile drawer — sits behind the content panel at z-index 1 */}
          <MobileDrawer
            open={drawerOpen}
            onClose={closeDrawer}
            activeItem={activeNavItem}
          />

          {/* Content panel — absolute on mobile (slides right), static on desktop */}
          <div
            className="absolute inset-0 md:relative md:inset-auto md:flex-1 flex flex-col bg-surface overflow-hidden"
            style={contentPanelStyle}
          >
            {/* Caustic and Bubbles live inside the panel so the transform context keeps them aligned */}
            <Caustic />
            {bubblesEnabled && <Bubbles />}

            <main className="flex-1 overflow-y-auto relative md:pb-0">
              {/* Scrim — covers content while drawer is open, tap to close */}
              {drawerOpen && (
                <div
                  className="md:hidden absolute inset-0 cursor-pointer"
                  style={{ zIndex: 50, background: 'rgba(7,13,31,0.35)' }}
                  onClick={closeDrawer}
                />
              )}
              <Outlet />
            </main>

            <MobileBottomBar
              open={drawerOpen}
              onToggle={() => setDrawerOpen(v => !v)}
              activeItem={activeNavItem}
            />
          </div>
        </div>
      </div>
    </PageThemeContext.Provider>
  )
}

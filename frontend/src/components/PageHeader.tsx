import { useState } from 'react'
import { Filter, ChevronDown, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { BottomSheet } from '@/components/BottomSheet'
import type { ReactNode, ElementType } from 'react'

export interface PageHeaderTab {
  key: string
  label: string
  icon?: ElementType
}

export interface PageHeaderAction {
  label: string
  icon?: ElementType
  onClick: () => void
  active?: boolean
}

interface PageHeaderProps {
  subtitle: string
  title: string
  action?: PageHeaderAction | null
  tabs?: PageHeaderTab[]
  activeTab?: string
  onTabChange?: (key: string) => void
  filterBar?: ReactNode
  filterActive?: boolean
  filterCount?: number
  onClearFilters?: () => void
  statusPills?: ReactNode
  maxWidth?: string
}

export function PageHeader({
  subtitle,
  title,
  action,
  tabs,
  activeTab,
  onTabChange,
  filterBar,
  filterActive,
  filterCount,
  onClearFilters,
  statusPills,
  maxWidth = 'max-w-7xl',
}: PageHeaderProps) {
  const [filterSheetOpen, setFilterSheetOpen] = useState(false)
  const hasTabs = tabs && tabs.length > 0
  const mobileSegmented = hasTabs && tabs.length <= 4
  const mobileSelect = hasTabs && tabs.length > 4

  return (
    <div>
      {/* ── Title row ── */}
      <div className={cn('px-6 md:px-8 pt-6 md:pt-8 mx-auto', maxWidth)}>
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs text-primary uppercase tracking-widest mb-1.5 font-medium">
              {subtitle}
            </p>
            <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
              {title}
            </h1>
          </div>

          {/* Desktop action — hidden on mobile (FAB below handles it) */}
          {action && (
            <button
              onClick={action.onClick}
              className={cn(
                'hidden md:flex shrink-0 items-center gap-2 px-5 py-2.5 rounded-full text-sm font-semibold transition-fluid',
                action.active
                  ? 'bg-primary/20 text-primary'
                  : 'bg-primary text-on-primary hover:brightness-110',
              )}
              style={!action.active ? { boxShadow: '0 0 20px rgba(58,223,250,0.25)' } : undefined}
            >
              {action.icon && <action.icon size={16} />}
              {action.label}
            </button>
          )}
        </div>

        {/* ── Desktop: underline tabs ── */}
        {hasTabs && (
          <div className="hidden md:block mt-5">
            <div className="flex gap-0 border-b border-outline-variant/20">
              {tabs.map((tab) => {
                const isActive = tab.key === activeTab
                const Icon = tab.icon
                return (
                  <button
                    key={tab.key}
                    onClick={() => onTabChange?.(tab.key)}
                    className={cn(
                      'flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-fluid',
                      isActive
                        ? 'text-primary border-primary'
                        : 'text-on-surface-dim border-transparent hover:text-on-surface hover:border-outline-variant/40',
                    )}
                  >
                    {Icon && <Icon size={13} strokeWidth={2} />}
                    {tab.label}
                  </button>
                )
              })}
            </div>
          </div>
        )}

        {/* ── Mobile: segmented control (2–4 tabs) ── */}
        {mobileSegmented && (
          <div className="md:hidden mt-4">
            <div className="flex bg-surface-container rounded-xl p-1 gap-1">
              {tabs.map((tab) => {
                const isActive = tab.key === activeTab
                const Icon = tab.icon
                return (
                  <button
                    key={tab.key}
                    onClick={() => onTabChange?.(tab.key)}
                    className={cn(
                      'flex-1 flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg text-sm font-medium transition-fluid',
                      isActive
                        ? 'bg-surface-container-high text-primary'
                        : 'text-on-surface-dim',
                    )}
                  >
                    {Icon && <Icon size={13} />}
                    {tab.label}
                  </button>
                )
              })}
            </div>
          </div>
        )}

        {/* ── Mobile: select dropdown (5+ tabs) ── */}
        {mobileSelect && (
          <div className="md:hidden mt-4 relative">
            <select
              value={activeTab}
              onChange={(e) => onTabChange?.(e.target.value)}
              className="w-full appearance-none bg-surface-container-high text-on-surface text-base font-semibold uppercase tracking-wider rounded-xl pl-4 pr-10 py-3 outline-none focus:ring-1 focus:ring-primary/50 cursor-pointer"
            >
              {tabs.map((tab) => (
                <option key={tab.key} value={tab.key}>{tab.label}</option>
              ))}
            </select>
            <div className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-on-surface-dim">
              <ChevronDown size={14} />
            </div>
          </div>
        )}

        {/* ── Mobile: Filters button or inline single filter ── */}
        {filterBar && (
          <div className="md:hidden mt-3">
            {filterCount === 1 ? (
              <div className="flex items-center gap-2">
                {filterBar}
                {filterActive && onClearFilters && (
                  <button
                    onClick={onClearFilters}
                    className="flex items-center gap-1 text-xs text-on-surface-dim hover:text-primary transition-fluid"
                  >
                    <X size={11} />
                    Clear
                  </button>
                )}
              </div>
            ) : (
              <button
                onClick={() => setFilterSheetOpen(true)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-fluid',
                  filterActive
                    ? 'bg-primary/15 text-primary'
                    : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high',
                )}
              >
                <Filter size={13} />
                Filters
                {filterActive && <span className="w-1.5 h-1.5 rounded-full bg-primary shrink-0" />}
              </button>
            )}
          </div>
        )}

        {/* ── Status pills ── */}
        {statusPills && (
          <div className="mt-3 flex flex-wrap gap-2">
            {statusPills}
          </div>
        )}
      </div>

      {/* ── Desktop filter bar (full-width darker strip) ── */}
      {filterBar && (
        <>
          {/* Divider — only when no tabs (tabs already have border-b) */}
          {!hasTabs && (
            <div className={cn('mx-auto px-6 md:px-8 mt-4', maxWidth)}>
              <div className="border-b border-outline-variant/20" />
            </div>
          )}
          <div className="hidden md:block bg-surface-container mt-0">
            <div className={cn('px-6 md:px-8 mx-auto py-2.5', maxWidth)}>
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-1.5 text-xs text-on-surface-faint shrink-0">
                  <Filter size={11} />
                  <span className="uppercase tracking-wider font-medium">Filter</span>
                </div>
                <div className="w-px h-3.5 bg-outline-variant/30 shrink-0" />
                <div className="flex items-center gap-1.5 flex-wrap min-w-0 flex-1">
                  {filterBar}
                </div>
                {filterActive && onClearFilters && (
                  <button
                    onClick={onClearFilters}
                    className="ml-auto flex items-center gap-1 text-xs text-on-surface-dim hover:text-primary transition-fluid shrink-0"
                  >
                    <X size={11} />
                    Clear filters
                  </button>
                )}
              </div>
            </div>
          </div>
        </>
      )}

      {/* ── Mobile: action FAB ── */}
      {action && (
        <button
          onClick={action.onClick}
          className={cn(
            'md:hidden fixed right-4 z-40 flex items-center gap-2 px-5 py-3 rounded-full text-sm font-semibold transition-fluid',
            action.active
              ? 'bg-primary/20 text-primary'
              : 'bg-primary text-on-primary',
          )}
          style={{
            bottom: '88px',
            boxShadow: action.active ? undefined : '0 4px 24px rgba(58,223,250,0.4)',
          }}
        >
          {action.icon && <action.icon size={16} />}
          {action.label}
        </button>
      )}

      {/* ── Mobile filter bottom sheet ── */}
      {filterBar && filterCount !== 1 && (
        <BottomSheet
          open={filterSheetOpen}
          onClose={() => setFilterSheetOpen(false)}
          title="Filters"
        >
          <div className="flex flex-col gap-4">
            {filterBar}
            {filterActive && onClearFilters && (
              <button
                onClick={() => { onClearFilters(); setFilterSheetOpen(false) }}
                className="mt-2 flex items-center justify-center gap-1.5 w-full py-2.5 rounded-xl text-sm font-medium text-on-surface-dim bg-surface-container hover:bg-surface-container-high hover:text-primary transition-fluid"
              >
                <X size={13} />
                Clear filters
              </button>
            )}
          </div>
        </BottomSheet>
      )}
    </div>
  )
}

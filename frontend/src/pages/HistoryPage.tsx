import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Clock, ScrollText, Activity, Plus } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import History, { INTERVALS } from '@/pages/History'
import type { HistoryRange } from '@/pages/History'
import Journal, { FullTimeline, KIND_LABELS, KIND_COLORS } from '@/pages/Journal'
import { PageHeader } from '@/components/PageHeader'
import { ProbeSelector, MeasurementSelector } from '@/components/ProbeSelector'
import { TimeRangePicker } from '@/components/TimeRangePicker'
import { useAuditEvents } from '@/hooks/useEvents'
import type { JournalCategory, JournalSentiment } from '@/api/client'

const CATEGORY_LABELS: Record<JournalCategory, string> = {
  observation: 'Observation',
  maintenance: 'Maintenance',
  event: 'Event',
  milestone: 'Milestone',
}

const SENTIMENT_LABELS: Record<JournalSentiment, string> = {
  good: 'Good',
  neutral: 'Neutral',
  bad: 'Bad',
  critical: 'Critical',
}

const CATEGORY_COLORS: Record<JournalCategory, string> = {
  observation: 'text-primary bg-primary/10',
  maintenance: 'text-on-surface-dim bg-surface-container-high',
  event: 'text-violet-400 bg-violet-400/10',
  milestone: 'text-amber-400 bg-amber-400/10',
}

const SENTIMENT_COLORS: Record<JournalSentiment, string> = {
  good: 'text-secondary bg-secondary/10',
  neutral: 'text-on-surface-dim bg-surface-container-high',
  bad: 'text-tertiary bg-tertiary/10',
  critical: 'text-red-400 bg-red-400/10',
}

export default function HistoryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'journal'
  usePageTitle('Log')

  // Journal state
  const [showEntryForm, setShowEntryForm] = useState(false)
  const [categoryFilter, setCategoryFilter] = useState<JournalCategory | ''>('')
  const [sentimentFilter, setSentimentFilter] = useState<JournalSentiment | ''>('')

  // Telemetry state
  const [selectedProbes, setSelectedProbes] = useState<string[]>([])
  const [selectedMeas, setSelectedMeas] = useState<string[]>([])
  const [range, setRange] = useState<HistoryRange>(() => {
    const now = new Date()
    return { from: new Date(now.getTime() - 86400000), to: now }
  })
  const [telemetryInterval, setTelemetryInterval] = useState('')

  // Timeline state
  const [kindFilter, setKindFilter] = useState('')
  const { data: auditData } = useAuditEvents({ limit: 100 })
  const allKinds = Array.from(new Set((auditData?.events ?? []).map(e => e.kind))).sort()

  function setTab(t: string) {
    setSearchParams({ tab: t }, { replace: true })
    setShowEntryForm(false)
  }

  const journalFilterBar = (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-1.5">
        <button onClick={() => setCategoryFilter('')} className={cn('px-3 py-1.5 rounded-full text-xs font-medium transition-fluid', categoryFilter === '' ? 'bg-primary/20 text-primary' : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high')}>All</button>
        {(Object.keys(CATEGORY_LABELS) as JournalCategory[]).map(c => (
          <button key={c} onClick={() => setCategoryFilter(categoryFilter === c ? '' : c)} className={cn('px-3 py-1.5 rounded-full text-xs font-medium transition-fluid', categoryFilter === c ? CATEGORY_COLORS[c] : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high')}>{CATEGORY_LABELS[c]}</button>
        ))}
      </div>
      <div className="flex flex-wrap gap-1.5">
        <button onClick={() => setSentimentFilter('')} className={cn('px-3 py-1.5 rounded-full text-xs font-medium transition-fluid', sentimentFilter === '' ? 'bg-primary/20 text-primary' : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high')}>Any</button>
        {(Object.keys(SENTIMENT_LABELS) as JournalSentiment[]).map(s => (
          <button key={s} onClick={() => setSentimentFilter(sentimentFilter === s ? '' : s)} className={cn('px-3 py-1.5 rounded-full text-xs font-medium transition-fluid', sentimentFilter === s ? SENTIMENT_COLORS[s] : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high')}>{SENTIMENT_LABELS[s]}</button>
        ))}
      </div>
    </div>
  )

  const telemetryFilterBar = (
    <div className="flex flex-wrap gap-x-4 gap-y-3 w-full">
      <div className="flex flex-wrap gap-2 items-start">
        <ProbeSelector selected={selectedProbes} onChange={setSelectedProbes} />
        <MeasurementSelector selected={selectedMeas} onChange={setSelectedMeas} colorOffset={selectedProbes.length} />
      </div>
      <TimeRangePicker value={range} onChange={setRange} />
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-xs text-on-surface-faint uppercase tracking-wider mr-1">Interval</span>
        {INTERVALS.map(int => (
          <button
            key={int.value}
            onClick={() => setTelemetryInterval(int.value)}
            className={cn(
              'px-2.5 py-1 rounded-full text-xs font-medium transition-fluid',
              telemetryInterval === int.value
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high',
            )}
          >
            {int.label}
          </button>
        ))}
      </div>
    </div>
  )

  const timelineFilterBar = (
    <div className="flex flex-wrap gap-1.5">
      <button
        onClick={() => setKindFilter('')}
        className={cn('px-3 py-1.5 rounded-full text-xs font-medium transition-fluid', kindFilter === '' ? 'bg-primary/20 text-primary' : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high')}
      >
        All events
      </button>
      {allKinds.map(k => (
        <button
          key={k}
          onClick={() => setKindFilter(kindFilter === k ? '' : k)}
          className={cn('px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
            kindFilter === k ? (KIND_COLORS[k] ?? 'bg-primary/20 text-primary') : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high'
          )}
        >
          {KIND_LABELS[k] ?? k}
        </button>
      ))}
    </div>
  )

  function getFilterBar() {
    if (tab === 'journal') return journalFilterBar
    if (tab === 'telemetry') return telemetryFilterBar
    if (tab === 'timeline') return allKinds.length > 0 ? timelineFilterBar : undefined
    return undefined
  }

  function getFilterActive() {
    if (tab === 'journal') return !!(categoryFilter || sentimentFilter)
    if (tab === 'telemetry') return selectedProbes.length > 0 || selectedMeas.length > 0
    if (tab === 'timeline') return !!kindFilter
    return false
  }

  return (
    <div>
      <PageHeader
        subtitle="Telemetry Archive and Log"
        title="Aquatic History"
        tabs={[
          { key: 'journal', label: 'Journal', icon: ScrollText },
          { key: 'telemetry', label: 'Telemetry', icon: Clock },
          { key: 'timeline', label: 'Full Timeline', icon: Activity },
        ]}
        activeTab={tab}
        onTabChange={setTab}
        action={tab === 'journal'
          ? { label: 'New Entry', icon: Plus, onClick: () => setShowEntryForm((v) => !v), active: showEntryForm }
          : null
        }
        filterBar={getFilterBar()}
        filterActive={getFilterActive()}
        maxWidth="max-w-7xl"
      />

      {tab === 'telemetry' && (
        <History
          hideHeader
          selectedProbes={selectedProbes}
          onSelectedProbesChange={setSelectedProbes}
          selectedMeas={selectedMeas}
          onSelectedMeasChange={setSelectedMeas}
          range={range}
          onRangeChange={setRange}
          interval={telemetryInterval}
          onIntervalChange={setTelemetryInterval}
        />
      )}
      {tab === 'journal' && (
        <Journal
          hideHeader
          categoryFilter={categoryFilter}
          onCategoryFilterChange={setCategoryFilter}
          sentimentFilter={sentimentFilter}
          onSentimentFilterChange={setSentimentFilter}
          showForm={showEntryForm}
          onShowFormChange={setShowEntryForm}
        />
      )}
      {tab === 'timeline' && (
        <div className="px-6 md:px-8 pt-6 pb-8 max-w-7xl mx-auto">
          <FullTimeline
            kindFilter={kindFilter}
            onKindFilterChange={setKindFilter}
            allKinds={allKinds}
          />
        </div>
      )}
    </div>
  )
}

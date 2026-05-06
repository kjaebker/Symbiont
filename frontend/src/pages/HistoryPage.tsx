import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Clock, ScrollText, Activity, Plus } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import History, { INTERVALS } from '@/pages/History'
import type { HistoryRange } from '@/pages/History'
import Journal, { FullTimeline, KIND_LABELS } from '@/pages/Journal'
import { PageHeader } from '@/components/PageHeader'
import { ProbeSelector, MeasurementSelector } from '@/components/ProbeSelector'
import { TimeRangePicker } from '@/components/TimeRangePicker'
import { FilterDropdown } from '@/components/FilterDropdown'
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
    <div className="flex flex-wrap gap-2">
      <FilterDropdown
        label="Category"
        value={categoryFilter}
        options={[
          { value: '', label: 'All categories' },
          ...(Object.keys(CATEGORY_LABELS) as JournalCategory[]).map((c) => ({ value: c, label: CATEGORY_LABELS[c] })),
        ]}
        onChange={(v) => setCategoryFilter(v as JournalCategory | '')}
      />
      <FilterDropdown
        label="Sentiment"
        value={sentimentFilter}
        options={[
          { value: '', label: 'Any sentiment' },
          ...(Object.keys(SENTIMENT_LABELS) as JournalSentiment[]).map((s) => ({ value: s, label: SENTIMENT_LABELS[s] })),
        ]}
        onChange={(v) => setSentimentFilter(v as JournalSentiment | '')}
      />
    </div>
  )

  const telemetryFilterBar = (
    <div className="flex flex-wrap gap-x-4 gap-y-3 w-full">
      <div className="flex flex-wrap gap-2 items-start">
        <ProbeSelector selected={selectedProbes} onChange={setSelectedProbes} />
        <MeasurementSelector selected={selectedMeas} onChange={setSelectedMeas} colorOffset={selectedProbes.length} />
      </div>
      <TimeRangePicker value={range} onChange={setRange} />
      <FilterDropdown
        label="Interval"
        value={telemetryInterval}
        options={INTERVALS.map((int) => ({ value: int.value, label: int.label }))}
        onChange={setTelemetryInterval}
      />
    </div>
  )

  const timelineFilterBar = allKinds.length > 0 ? (
    <FilterDropdown
      label="All events"
      value={kindFilter}
      options={[
        { value: '', label: 'All events' },
        ...allKinds.map((k) => ({ value: k, label: KIND_LABELS[k] ?? k })),
      ]}
      onChange={setKindFilter}
    />
  ) : undefined

  function getFilterBar() {
    if (tab === 'journal') return journalFilterBar
    if (tab === 'telemetry') return telemetryFilterBar
    if (tab === 'timeline') return timelineFilterBar
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

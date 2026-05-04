import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Clock, ScrollText, Activity, Plus } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import History from '@/pages/History'
import Journal, { FullTimeline } from '@/pages/Journal'
import { PageHeader } from '@/components/PageHeader'
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

  const [showEntryForm, setShowEntryForm] = useState(false)
  const [categoryFilter, setCategoryFilter] = useState<JournalCategory | ''>('')
  const [sentimentFilter, setSentimentFilter] = useState<JournalSentiment | ''>('')

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
        filterBar={tab === 'journal' ? journalFilterBar : undefined}
        filterActive={!!(categoryFilter || sentimentFilter)}
        maxWidth="max-w-7xl"
      />

      {tab === 'telemetry' && <History hideHeader />}
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
          <FullTimeline />
        </div>
      )}
    </div>
  )
}

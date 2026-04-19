import { useSearchParams } from 'react-router-dom'
import { Clock, ScrollText } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import History from '@/pages/History'
import Journal from '@/pages/Journal'

export default function HistoryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'telemetry'
  usePageTitle('History')

  function setTab(t: string) {
    setSearchParams({ tab: t }, { replace: true })
  }

  return (
    <div>
      <div className="px-6 md:px-8 pt-6 max-w-6xl mx-auto flex gap-1.5">
        <button
          onClick={() => setTab('telemetry')}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-fluid',
            tab === 'telemetry'
              ? 'bg-primary/20 text-primary'
              : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
          )}
        >
          <Clock size={13} />
          Telemetry
        </button>
        <button
          onClick={() => setTab('journal')}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-fluid',
            tab === 'journal'
              ? 'bg-primary/20 text-primary'
              : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
          )}
        >
          <ScrollText size={13} />
          Journal
        </button>
      </div>

      {tab === 'telemetry' ? (
        <History />
      ) : (
        <Journal />
      )}
    </div>
  )
}

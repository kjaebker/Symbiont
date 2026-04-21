import { useSearchParams } from 'react-router-dom'
import { Clock, ScrollText, Activity } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import History from '@/pages/History'
import Journal, { FullTimeline } from '@/pages/Journal'

export default function HistoryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'journal'
  usePageTitle('Log')

  function setTab(t: string) {
    setSearchParams({ tab: t }, { replace: true })
  }

  return (
    <div>
      <div className="px-6 md:px-8 pt-8 max-w-7xl mx-auto">
        <p className="text-xs text-primary uppercase tracking-widest mb-2">Telemetry Archive and Log</p>
        <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight mb-6">Aquatic History</h1>
        <div className="flex gap-1.5">
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
            onClick={() => setTab('timeline')}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-fluid',
              tab === 'timeline'
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
            )}
          >
            <Activity size={13} />
            Full Timeline
          </button>
        </div>
      </div>

      {tab === 'telemetry' && <History hideHeader />}
      {tab === 'journal' && <Journal hideHeader />}
      {tab === 'timeline' && (
        <div className="px-6 md:px-8 pt-6 pb-8 max-w-7xl mx-auto">
          <FullTimeline />
        </div>
      )}
    </div>
  )
}

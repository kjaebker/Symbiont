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
    <div className="min-h-full">
      {/* Tab bar */}
      <div className="sticky top-0 z-10 bg-surface/90 backdrop-blur-md border-b border-surface-container">
        <div className="max-w-6xl mx-auto px-6 pt-6 pb-0">
          <div className="flex items-center gap-1 mb-0">
            <h1 className="text-xl font-bold text-on-surface mr-4">History</h1>
            <div className="flex gap-1">
              <button
                onClick={() => setTab('telemetry')}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-t-xl text-sm font-medium transition-fluid border-b-2',
                  tab === 'telemetry'
                    ? 'text-primary border-primary bg-primary/5'
                    : 'text-on-surface-dim border-transparent hover:text-on-surface',
                )}
              >
                <Clock size={15} />
                Telemetry
              </button>
              <button
                onClick={() => setTab('journal')}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-t-xl text-sm font-medium transition-fluid border-b-2',
                  tab === 'journal'
                    ? 'text-primary border-primary bg-primary/5'
                    : 'text-on-surface-dim border-transparent hover:text-on-surface',
                )}
              >
                <ScrollText size={15} />
                Journal
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Tab content */}
      {tab === 'telemetry' ? (
        <History />
      ) : (
        <Journal />
      )}
    </div>
  )
}

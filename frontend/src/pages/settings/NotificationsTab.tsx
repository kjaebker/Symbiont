import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Plus,Trash2,Bell } from 'lucide-react'
import { useNotificationTargets,useUpsertNotificationTarget,useDeleteNotificationTarget,useTestNotifications } from '@/hooks/useNotifications'
import type { NotificationTarget } from '@/api/types'
import { LoadingState,EmptyState } from './_shared'

// =============================================================================
// Notifications Tab
// =============================================================================

export default function NotificationsTab() {
  const { data, isLoading } = useNotificationTargets()
  const upsertMutation = useUpsertNotificationTarget()
  const deleteMutation = useDeleteNotificationTarget()
  const testMutation = useTestNotifications()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [url, setUrl] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  const targets = data?.targets ?? []

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!label.trim() || !url.trim()) return
    upsertMutation.mutate(
      { type: 'ntfy', label: label.trim(), config: url.trim(), enabled: true },
      {
        onSuccess: () => {
          setLabel('')
          setUrl('')
          setShowForm(false)
        },
      },
    )
  }

  function handleToggle(t: NotificationTarget) {
    upsertMutation.mutate({ id: t.id, type: t.type, label: t.label, config: t.config, enabled: !t.enabled })
  }

  function handleDelete(id: number) {
    deleteMutation.mutate(id, {
      onSuccess: () => setDeleteConfirm(null),
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-4 pt-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          ntfy.sh Targets
        </span>
        <div className="flex items-center gap-2">
          {targets.some((t) => t.enabled) && (
            <button
              onClick={() => testMutation.mutate()}
              disabled={testMutation.isPending}
              className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer disabled:opacity-50"
            >
              <Bell size={14} />
              Send Test
            </button>
          )}
          {!showForm && (
            <button
              onClick={() => setShowForm(true)}
              className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
            >
              <Plus size={14} />
              Add Target
            </button>
          )}
        </div>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="flex items-center gap-2 px-4 flex-wrap">
          <input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="Label (e.g. Phone)"
            className="flex-1 min-w-32 bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
            autoFocus
          />
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://ntfy.sh/your-topic"
            className="flex-[3] min-w-48 bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
          />
          <button
            type="submit"
            disabled={upsertMutation.isPending || !label.trim() || !url.trim()}
            className="px-3 py-2 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
          >
            Add
          </button>
          <button
            type="button"
            onClick={() => { setShowForm(false); setLabel(''); setUrl('') }}
            className="px-3 py-2 rounded-xl text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
          >
            Cancel
          </button>
        </form>
      )}

      {testMutation.isSuccess && (
        <div className="mx-4 space-y-1">
          {testMutation.data?.results.map((r) => (
            <div
              key={r.label}
              className={cn('rounded-xl px-4 py-2', r.success ? 'bg-secondary/10' : 'bg-tertiary/10')}
            >
              <p className={cn('text-xs font-medium', r.success ? 'text-secondary' : 'text-tertiary')}>
                {r.label}: {r.success ? 'Test notification sent.' : `Failed — ${r.error}`}
              </p>
            </div>
          ))}
        </div>
      )}

      {testMutation.isError && (
        <div className="mx-4 bg-tertiary/10 rounded-xl px-4 py-2">
          <p className="text-xs text-tertiary font-medium">
            {(testMutation.error as Error)?.message ?? 'Test failed. Check your ntfy topic URL.'}
          </p>
        </div>
      )}

      {isLoading ? (
        <LoadingState label="Loading notification targets..." />
      ) : targets.length === 0 ? (
        <EmptyState
          icon={<Bell size={32} />}
          message="No notification targets configured. Add an ntfy.sh topic URL to get alerted on your phone."
        />
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {targets.map((t) => (
              <div key={t.id} className="bg-surface-container-high rounded-2xl p-4 space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-semibold text-on-surface">{t.label}</span>
                  <button
                    onClick={() => handleToggle(t)}
                    disabled={upsertMutation.isPending}
                    className={cn(
                      'px-2.5 py-1 rounded-full text-xs font-medium uppercase tracking-wider transition-fluid cursor-pointer disabled:opacity-50 shrink-0',
                      t.enabled
                        ? 'bg-secondary/15 text-secondary hover:bg-secondary/25'
                        : 'bg-surface-container-highest text-on-surface-faint hover:bg-surface-container-highest/80',
                    )}
                  >
                    {t.enabled ? 'Enabled' : 'Disabled'}
                  </button>
                </div>
                <p className="text-xs text-on-surface-dim font-mono break-all leading-relaxed">{t.config}</p>
                <div className="flex items-center justify-end">
                  {deleteConfirm === t.id ? (
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleDelete(t.id)}
                        className="px-2 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer"
                      >
                        Delete
                      </button>
                      <button
                        onClick={() => setDeleteConfirm(null)}
                        className="px-2 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={() => setDeleteConfirm(t.id)}
                      className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
                    >
                      <Trash2 size={14} />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
          {/* Desktop table */}
          <div className="hidden sm:block overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-container-high/50">
                  {['Label', 'URL', 'Status', ''].map((h) => (
                    <th
                      key={h}
                      className={cn(
                        'py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest',
                        h === '' ? 'text-right' : 'text-left',
                      )}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {targets.map((t) => (
                  <tr key={t.id} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-3 px-4 text-sm font-medium text-on-surface">{t.label}</td>
                    <td className="py-3 px-4 text-sm text-on-surface-dim font-mono truncate max-w-xs">{t.config}</td>
                    <td className="py-3 px-4">
                      <button
                        onClick={() => handleToggle(t)}
                        disabled={upsertMutation.isPending}
                        className={cn(
                          'px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider transition-fluid cursor-pointer disabled:opacity-50',
                          t.enabled
                            ? 'bg-secondary/15 text-secondary hover:bg-secondary/25'
                            : 'bg-surface-container-highest text-on-surface-faint hover:bg-surface-container-highest/80',
                        )}
                      >
                        {t.enabled ? 'Enabled' : 'Disabled'}
                      </button>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center justify-end">
                        {deleteConfirm === t.id ? (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleDelete(t.id)}
                              className="px-2 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer"
                            >
                              Delete
                            </button>
                            <button
                              onClick={() => setDeleteConfirm(null)}
                              className="px-2 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <button
                            onClick={() => setDeleteConfirm(t.id)}
                            className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer"
                          >
                            <Trash2 size={14} />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}


import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Plus,Trash2,Bell } from 'lucide-react'
import { useNotificationTargets,useUpsertNotificationTarget,useDeleteNotificationTarget,useTestNotifications } from '@/hooks/useNotifications'
import type { NotificationTarget } from '@/api/types'
import { Button, Input, SectionHeader, LoadingState, EmptyState, labelClass } from '@/components/ui'
import { ConfirmButton } from '@/components/ConfirmButton'

// =============================================================================
// Notifications Tab
// =============================================================================

/**
 * An ntfy topic URL is a secret: anyone who knows the path can read and send
 * to the topic. Show the host and only the last characters of the path.
 */
function maskTargetUrl(raw: string): string {
  try {
    const u = new URL(raw)
    const secret = u.pathname.replace(/^\//, '') + u.search
    if (!secret) return u.host
    return `${u.host}/••••${secret.length > 6 ? secret.slice(-3) : ''}`
  } catch {
    return '••••'
  }
}

export default function NotificationsTab() {
  const { data, isLoading } = useNotificationTargets()
  const upsertMutation = useUpsertNotificationTarget()
  const deleteMutation = useDeleteNotificationTarget()
  const testMutation = useTestNotifications()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [url, setUrl] = useState('')

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
    deleteMutation.mutate(id)
  }

  return (
    <div className="space-y-4">
      <SectionHeader
        title="ntfy.sh Targets"
        className="px-4 pt-2"
        action={
          <div className="flex items-center gap-2">
            {targets.some((t) => t.enabled) && (
              <Button
                size="sm"
                icon={Bell}
                loading={testMutation.isPending}
                onClick={() => testMutation.mutate()}
              >
                Send Test
              </Button>
            )}
            {!showForm && (
              <Button variant="tonal" size="sm" icon={Plus} onClick={() => setShowForm(true)}>
                Add Target
              </Button>
            )}
          </div>
        }
      />

      {showForm && (
        <form onSubmit={handleCreate} className="flex items-center gap-2 px-4 flex-wrap">
          <Input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="Label (e.g. Phone)"
            className="flex-1 min-w-32 w-auto"
            autoFocus
          />
          <Input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://ntfy.sh/your-topic"
            className="flex-[3] min-w-48 w-auto"
          />
          <Button
            type="submit"
            variant="primary"
            disabled={upsertMutation.isPending || !label.trim() || !url.trim()}
          >
            Add
          </Button>
          <Button onClick={() => { setShowForm(false); setLabel(''); setUrl('') }}>
            Cancel
          </Button>
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
                <p className="text-xs text-on-surface-dim font-mono break-all leading-relaxed">{maskTargetUrl(t.config)}</p>
                <div className="flex items-center justify-end">
                  <ConfirmButton
                    onConfirm={() => handleDelete(t.id)}
                    title="Delete target"
                    className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                  >
                    <Trash2 size={14} />
                  </ConfirmButton>
                </div>
              </div>
            ))}
          </div>
          {/* Desktop table */}
          <div className="hidden sm:block overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-container-high/50">
                  {['Label', 'URL', 'Status', ''].map((h, i) => (
                    <th
                      key={i}
                      className={cn('py-3 px-4', labelClass, h === '' ? 'text-right' : 'text-left')}
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
                    <td className="py-3 px-4 text-sm text-on-surface-dim font-mono truncate max-w-xs">{maskTargetUrl(t.config)}</td>
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
                        <ConfirmButton
                    onConfirm={() => handleDelete(t.id)}
                    title="Delete target"
                    className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                  >
                    <Trash2 size={14} />
                  </ConfirmButton>
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


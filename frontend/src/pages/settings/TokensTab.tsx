import { useState } from 'react'
import { cn,relativeTime } from '@/lib/utils'
import { Settings as SettingsIcon,Plus,Trash2,Copy,Check } from 'lucide-react'
import { useTokens,useCreateToken,useUpdateTokenScope,useRevokeToken } from '@/hooks/useSettings'
import { LoadingState,EmptyState } from './_shared'

// =============================================================================
// Tokens Tab
// =============================================================================

export default function TokensTab() {
  const { data, isLoading } = useTokens()
  const createMutation = useCreateToken()
  const updateScopeMutation = useUpdateTokenScope()
  const revokeMutation = useRevokeToken()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [scope, setScope] = useState<'read' | 'write' | 'control' | 'admin'>('admin')
  const [newToken, setNewToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [revokeConfirm, setRevokeConfirm] = useState<number | null>(null)

  const tokens = data?.tokens ?? []

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!label.trim()) return
    createMutation.mutate({ label: label.trim(), scope }, {
      onSuccess: (data) => {
        setNewToken(data.token)
        setLabel('')
        setScope('admin')
        setShowForm(false)
      },
    })
  }

  function handleCopy(token: string) {
    navigator.clipboard.writeText(token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  function handleRevoke(id: number) {
    revokeMutation.mutate(id, {
      onSuccess: () => setRevokeConfirm(null),
    })
  }

  return (
    <div className="space-y-4">
      {newToken && (
        <div className="bg-secondary/10 rounded-2xl p-4 space-y-2">
          <p className="text-xs text-secondary uppercase tracking-widest font-medium">
            Token Created
          </p>
          <p className="text-xs text-on-surface-dim">
            Copy this token now. It will not be shown again.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-surface-container-high rounded-lg px-3 py-2 text-sm text-on-surface font-mono break-all">
              {newToken}
            </code>
            <button
              onClick={() => handleCopy(newToken)}
              className="p-2 rounded-lg text-on-surface-faint hover:text-secondary hover:bg-secondary/10 transition-fluid cursor-pointer"
            >
              {copied ? <Check size={16} /> : <Copy size={16} />}
            </button>
          </div>
          <button
            onClick={() => setNewToken(null)}
            className="text-xs text-on-surface-faint hover:text-on-surface transition-fluid cursor-pointer"
          >
            Dismiss
          </button>
        </div>
      )}

      <div className="flex items-center justify-between px-4 pt-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          API Tokens
        </span>
        {!showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
          >
            <Plus size={14} />
            Create Token
          </button>
        )}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="flex items-center gap-2 px-4">
          <input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="Token label (e.g. CLI, MCP)"
            className="flex-1 bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid"
            autoFocus
          />
          <select
            value={scope}
            onChange={(e) => setScope(e.target.value as 'read' | 'write' | 'control' | 'admin')}
            className="bg-surface-container-high text-on-surface text-base rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid cursor-pointer"
          >
            <option value="admin">admin</option>
            <option value="control">control</option>
            <option value="write">write</option>
            <option value="read">read</option>
          </select>
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="px-3 py-2 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
          >
            Create
          </button>
          <button
            type="button"
            onClick={() => {
              setShowForm(false)
              setLabel('')
              setScope('admin')
            }}
            className="px-3 py-2 rounded-xl text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
          >
            Cancel
          </button>
        </form>
      )}

      {isLoading ? (
        <LoadingState label="Loading tokens..." />
      ) : tokens.length === 0 ? (
        <EmptyState
          icon={<SettingsIcon size={32} />}
          message="No API tokens. Create one to authenticate CLI or MCP clients."
        />
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {tokens.map((t) => (
              <div key={t.id} className="bg-surface-container-high rounded-2xl p-4 space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <span className="text-sm font-semibold text-on-surface">{t.label}</span>
                    <div className="text-xs text-on-surface-faint font-mono mt-0.5">#{t.id}</div>
                  </div>
                  <select
                    value={t.scope}
                    onChange={(e) => updateScopeMutation.mutate({ id: t.id, scope: e.target.value })}
                    disabled={updateScopeMutation.isPending}
                    className="bg-surface-container-highest text-on-surface text-xs rounded-lg px-2 py-1 outline-none focus:ring-1 focus:ring-primary/30 cursor-pointer transition-fluid disabled:opacity-50 shrink-0"
                  >
                    <option value="admin">admin</option>
                    <option value="control">control</option>
                    <option value="write">write</option>
                    <option value="read">read</option>
                  </select>
                </div>
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-3 text-xs text-on-surface-faint">
                    <span>Created {relativeTime(t.created_at)}</span>
                    <span>Used {t.last_used ? relativeTime(t.last_used) : 'never'}</span>
                  </div>
                  {revokeConfirm === t.id ? (
                    <div className="flex items-center gap-1 shrink-0">
                      <button
                        onClick={() => handleRevoke(t.id)}
                        className="px-2 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer"
                      >
                        Revoke
                      </button>
                      <button
                        onClick={() => setRevokeConfirm(null)}
                        className="px-2 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={() => setRevokeConfirm(t.id)}
                      className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid cursor-pointer shrink-0"
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
                  {['ID', 'Label', 'Scope', 'Created', 'Last Used', ''].map((h) => (
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
                {tokens.map((t) => (
                  <tr key={t.id} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-3 px-4 text-sm text-on-surface-dim font-mono">{t.id}</td>
                    <td className="py-3 px-4 text-sm font-medium text-on-surface">{t.label}</td>
                    <td className="py-3 px-4">
                      <select
                        value={t.scope}
                        onChange={(e) => updateScopeMutation.mutate({ id: t.id, scope: e.target.value })}
                        disabled={updateScopeMutation.isPending}
                        className="bg-surface-container-highest text-on-surface text-xs rounded-lg px-2 py-1 outline-none focus:ring-1 focus:ring-primary/30 cursor-pointer transition-fluid disabled:opacity-50"
                      >
                        <option value="admin">admin</option>
                        <option value="control">control</option>
                        <option value="read">read</option>
                      </select>
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface-dim">
                      {relativeTime(t.created_at)}
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface-dim">
                      {t.last_used ? relativeTime(t.last_used) : 'Never'}
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center justify-end">
                        {revokeConfirm === t.id ? (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleRevoke(t.id)}
                              className="px-2 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer"
                            >
                              Revoke
                            </button>
                            <button
                              onClick={() => setRevokeConfirm(null)}
                              className="px-2 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <button
                            onClick={() => setRevokeConfirm(t.id)}
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


import { useState } from 'react'

import { Settings as SettingsIcon,Plus,Trash2,Copy,Check } from 'lucide-react'
import { useTokens,useCreateToken,useUpdateTokenScope,useRevokeToken } from '@/hooks/useSettings'
import { APIRequestError } from '@/api/_fetch'
import { Button, IconButton, Input, Select, SectionHeader, LoadingState, EmptyState, labelClass, TimeAgo } from '@/components/ui'
import { ConfirmButton } from '@/components/ConfirmButton'

// =============================================================================
// Tokens Tab
// =============================================================================

export default function TokensTab() {
  const { data, isLoading, isError, error } = useTokens()
  const createMutation = useCreateToken()
  const updateScopeMutation = useUpdateTokenScope()
  const revokeMutation = useRevokeToken()

  const [showForm, setShowForm] = useState(false)
  const [label, setLabel] = useState('')
  const [scope, setScope] = useState<'read' | 'write' | 'control' | 'admin'>('admin')
  const [newToken, setNewToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

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
    revokeMutation.mutate(id)
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
            <IconButton
              icon={copied ? Check : Copy}
              size={16}
              label="Copy token"
              tone="primary"
              onClick={() => handleCopy(newToken)}
              className="p-2"
            />
          </div>
          <Button variant="ghost" size="sm" onClick={() => setNewToken(null)}>
            Dismiss
          </Button>
        </div>
      )}

      <SectionHeader
        title="API Tokens"
        className="px-4 pt-2"
        action={!showForm && (
          <Button variant="tonal" size="sm" icon={Plus} onClick={() => setShowForm(true)}>
            Create Token
          </Button>
        )}
      />

      {showForm && (
        <form onSubmit={handleCreate} className="flex flex-wrap items-center gap-2 px-4">
          <Input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="Token label (e.g. CLI, MCP)"
            className="flex-1 min-w-48 w-auto"
            autoFocus
          />
          <Select
            value={scope}
            onChange={(e) => setScope(e.target.value as 'read' | 'write' | 'control' | 'admin')}
            className="w-auto"
          >
            <option value="admin">admin</option>
            <option value="control">control</option>
            <option value="write">write</option>
            <option value="read">read</option>
          </Select>
          <Button type="submit" variant="primary" loading={createMutation.isPending}>
            Create
          </Button>
          <Button
            onClick={() => {
              setShowForm(false)
              setLabel('')
              setScope('admin')
            }}
          >
            Cancel
          </Button>
        </form>
      )}

      {isLoading ? (
        <LoadingState label="Loading tokens..." />
      ) : isError ? (
        <EmptyState
          icon={<SettingsIcon size={32} />}
          message={error instanceof APIRequestError && error.status === 403
            ? 'This token is not allowed to manage API tokens. Sign in with an admin token to see them.'
            : `Couldn't load API tokens: ${error.message}`}
        />
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
                  <Select
                    value={t.scope}
                    onChange={(e) => updateScopeMutation.mutate({ id: t.id, scope: e.target.value })}
                    disabled={updateScopeMutation.isPending}
                    className="bg-surface-container-highest w-auto py-1 px-2 shrink-0"
                  >
                    <option value="admin">admin</option>
                    <option value="control">control</option>
                    <option value="write">write</option>
                    <option value="read">read</option>
                  </Select>
                </div>
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-3 text-xs text-on-surface-faint">
                    <span>Created <TimeAgo ts={t.created_at} /></span>
                    <span>Used {t.last_used ? <TimeAgo ts={t.last_used} /> : 'never'}</span>
                  </div>
                  <ConfirmButton
                    onConfirm={() => handleRevoke(t.id)}
                    confirmLabel="Revoke"
                    title="Revoke token"
                    className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid shrink-0"
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
                  {['ID', 'Label', 'Scope', 'Created', 'Last Used', ''].map((h, i) => (
                    <th
                      key={i}
                      className={`py-3 px-4 ${labelClass} ${h === '' ? 'text-right' : 'text-left'}`}
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
                      <Select
                        value={t.scope}
                        onChange={(e) => updateScopeMutation.mutate({ id: t.id, scope: e.target.value })}
                        disabled={updateScopeMutation.isPending}
                        className="bg-surface-container-highest w-auto py-1 px-2"
                      >
                        <option value="admin">admin</option>
                        <option value="control">control</option>
                        <option value="read">read</option>
                      </Select>
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface-dim">
                      <TimeAgo ts={t.created_at} />
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface-dim">
                      {t.last_used ? <TimeAgo ts={t.last_used} /> : 'Never'}
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center justify-end">
                        <ConfirmButton
                          onConfirm={() => handleRevoke(t.id)}
                          confirmLabel="Revoke"
                          title="Revoke token"
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

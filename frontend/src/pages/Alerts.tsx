import { useState } from 'react'
import { Plus, Pencil, Trash2, Bell } from 'lucide-react'
import { useAlerts, useCreateAlert, useUpdateAlert, useDeleteAlert } from '@/hooks/useAlerts'
import { usePageTitle } from '@/hooks/usePageTitle'
import { AlertRuleForm } from '@/components/AlertRuleForm'
import type { AlertRule } from '@/api/types'
import { cn } from '@/lib/utils'

const conditionLabels: Record<string, string> = {
  above: 'Above',
  below: 'Below',
  outside_range: 'Outside',
}

function formatThreshold(rule: AlertRule): string {
  if (rule.condition === 'above' && rule.threshold_high != null) {
    return `> ${rule.threshold_high}`
  }
  if (rule.condition === 'below' && rule.threshold_low != null) {
    return `< ${rule.threshold_low}`
  }
  if (rule.condition === 'outside_range' && rule.threshold_low != null && rule.threshold_high != null) {
    return `${rule.threshold_low} – ${rule.threshold_high}`
  }
  return '--'
}

export default function Alerts() {
  usePageTitle('Alerts')
  const { data, isLoading } = useAlerts()
  const createMutation = useCreateAlert()
  const updateMutation = useUpdateAlert()
  const deleteMutation = useDeleteAlert()

  const [formOpen, setFormOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | undefined>()
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  const rules = data?.rules ?? []

  function handleCreate(rule: Omit<AlertRule, 'id' | 'created_at'>) {
    createMutation.mutate(rule, {
      onSuccess: () => setFormOpen(false),
    })
  }

  function handleEdit(rule: Omit<AlertRule, 'id' | 'created_at'>) {
    if (!editingRule) return
    updateMutation.mutate(
      { id: editingRule.id, rule },
      { onSuccess: () => setEditingRule(undefined) },
    )
  }

  function handleToggle(rule: AlertRule) {
    updateMutation.mutate({
      id: rule.id,
      rule: { ...rule, enabled: !rule.enabled },
    })
  }

  function handleDelete(id: number) {
    deleteMutation.mutate(id, {
      onSuccess: () => setDeleteConfirm(null),
    })
  }

  return (
    <div className="p-6 md:p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs text-primary uppercase tracking-widest mb-2">
            Monitoring
          </p>
          <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
            Alert Rules
          </h1>
        </div>
        <button
          onClick={() => setFormOpen(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
        >
          <Plus size={16} />
          New Rule
        </button>
      </div>

      {/* Rules Table */}
      <div className="bg-surface-container rounded-2xl overflow-hidden">
        {isLoading ? (
          <div className="p-12 flex items-center justify-center">
            <div className="flex flex-col items-center gap-3">
              <div className="h-8 w-8 rounded-full bg-primary/20 animate-bio-pulse" />
              <span className="text-xs text-on-surface-faint uppercase tracking-widest">
                Loading alert rules...
              </span>
            </div>
          </div>
        ) : rules.length === 0 ? (
          <div className="p-12 flex flex-col items-center justify-center gap-3">
            <Bell size={32} className="text-on-surface-faint" />
            <span className="text-on-surface-dim text-sm text-center max-w-sm">
              No alert rules configured. Add one to get notified when parameters go out of range.
            </span>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-container-high/50">
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Probe
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Condition
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Threshold
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Severity
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Status
                  </th>
                  <th className="text-right py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr
                    key={rule.id}
                    className="transition-fluid hover:bg-surface-container-high/50"
                  >
                    <td className="py-3 px-4 text-sm font-medium text-on-surface">
                      {rule.probe_name}
                    </td>
                    <td className="py-3 px-4 text-xs text-on-surface-dim uppercase tracking-wider">
                      {conditionLabels[rule.condition] ?? rule.condition}
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface font-mono">
                      {formatThreshold(rule)}
                    </td>
                    <td className="py-3 px-4">
                      <span
                        className={cn(
                          'inline-block px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider',
                          rule.severity === 'critical'
                            ? 'bg-tertiary/15 text-tertiary'
                            : 'bg-amber-400/15 text-amber-400',
                        )}
                      >
                        {rule.severity}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <button
                        onClick={() => handleToggle(rule)}
                        className={cn(
                          'relative w-10 h-5 rounded-full transition-fluid cursor-pointer',
                          rule.enabled ? 'bg-primary' : 'bg-surface-container-highest',
                        )}
                      >
                        <span
                          className={cn(
                            'absolute top-0.5 h-4 w-4 rounded-full bg-on-surface transition-fluid',
                            rule.enabled ? 'left-5.5' : 'left-0.5',
                          )}
                        />
                      </button>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => setEditingRule(rule)}
                          className="p-1.5 rounded-lg text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high transition-fluid cursor-pointer"
                        >
                          <Pencil size={14} />
                        </button>
                        {deleteConfirm === rule.id ? (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleDelete(rule.id)}
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
                            onClick={() => setDeleteConfirm(rule.id)}
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
        )}
      </div>

      {/* Mutation errors */}
      {(createMutation.isError || updateMutation.isError || deleteMutation.isError) && (
        <div className="bg-tertiary/10 rounded-xl p-3 text-sm text-tertiary">
          {createMutation.isError && 'Failed to create alert rule. '}
          {updateMutation.isError && 'Failed to update alert rule. '}
          {deleteMutation.isError && 'Failed to delete alert rule. '}
          Please try again.
        </div>
      )}

      {/* Dialogs */}
      {formOpen && (
        <AlertRuleForm
          onSubmit={handleCreate}
          onClose={() => setFormOpen(false)}
        />
      )}
      {editingRule && (
        <AlertRuleForm
          rule={editingRule}
          onSubmit={handleEdit}
          onClose={() => setEditingRule(undefined)}
        />
      )}
    </div>
  )
}

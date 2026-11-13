import { useState } from 'react'
import { Plus, Pencil, Trash2, Bell, Activity } from 'lucide-react'
import { useAlerts, useCreateAlert, useUpdateAlert, useDeleteAlert, useAlertEvents } from '@/hooks/useAlerts'
import { usePageTitle } from '@/hooks/usePageTitle'
import { AlertRuleForm } from '@/components/AlertRuleForm'
import { PageHeader } from '@/components/PageHeader'
import type { AlertRule } from '@/api/types'
import { cn } from '@/lib/utils'
import { Card, Chip, EmptyState, IconButton, LoadingState, PageBody, SectionHeader, Toggle, labelClass } from '@/components/ui'
import { ConfirmButton } from '@/components/ConfirmButton'

const conditionLabels: Record<string, string> = {
  probe_range: 'Probe range',
  above: 'Above',
  below: 'Below',
  outside_range: 'Outside',
}

function formatThreshold(rule: AlertRule): string {
  if (rule.condition === 'probe_range') {
    if (!rule.range_configured) return 'No range set'
    const low = rule.range_low
    const high = rule.range_high
    if (low != null && high != null) return `${low} – ${high}`
    if (low != null) return `< ${low}`
    if (high != null) return `> ${high}`
    return 'OK state'
  }
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

/** A probe_range rule whose probe has no range can never fire. */
function isUnconfigured(rule: AlertRule): boolean {
  return rule.condition === 'probe_range' && !rule.range_configured
}

export default function Alerts() {
  usePageTitle('Alerts')
  const { data, isLoading } = useAlerts()
  const createMutation = useCreateAlert()
  const updateMutation = useUpdateAlert()
  const deleteMutation = useDeleteAlert()

  const [formOpen, setFormOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | undefined>()

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

  return (
    <div>
      <PageHeader
        subtitle="Monitoring"
        title="Alert Rules"
        action={{ label: 'New Rule', icon: Plus, onClick: () => setFormOpen(true) }}
        maxWidth="max-w-7xl"
      />

      <PageBody>
      {/* Rules Table */}
      {isLoading ? (
        <Card flush><LoadingState label="Loading alert rules..." /></Card>
      ) : rules.length === 0 ? (
        <Card flush>
          <EmptyState
            icon={<Bell size={32} />}
            message="No alert rules yet. Add one to get notified when a probe leaves its range."
          />
        </Card>
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {rules.map((rule) => (
              <div key={rule.id} className="bg-surface-container rounded-2xl px-4 py-3 space-y-2">
                <div className="flex items-start justify-between gap-2">
                  <div className={cn('transition-fluid', !rule.enabled && 'opacity-50')}>
                    <span className="text-sm font-medium text-on-surface">
                      {rule.probe_display_name ?? rule.probe_name}
                    </span>
                    <div className="flex items-center gap-2 mt-1 flex-wrap">
                      <span className="text-xs text-on-surface-dim uppercase tracking-wider">
                        {conditionLabels[rule.condition] ?? rule.condition}
                      </span>
                      <span className={cn('text-xs font-mono', isUnconfigured(rule) ? 'text-warning' : 'text-on-surface')}>{formatThreshold(rule)}</span>
                      <Chip tone={rule.severity === 'critical' ? 'critical' : 'warning'}>{rule.severity}</Chip>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Toggle
                      checked={rule.enabled}
                      onChange={() => handleToggle(rule)}
                      label={`${rule.enabled ? 'Disable' : 'Enable'} rule for ${rule.probe_display_name ?? rule.probe_name}`}
                    />
                    <IconButton icon={Pencil} label="Edit rule" onClick={() => setEditingRule(rule)} />
                    <ConfirmButton
                      onConfirm={() => deleteMutation.mutate(rule.id)}
                      title="Delete rule"
                      className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                    >
                      <Trash2 size={14} />
                    </ConfirmButton>
                  </div>
                </div>
              </div>
            ))}
          </div>
          {/* Desktop table */}
          <Card flush className="hidden sm:block">
              <table className="w-full">
                <thead>
                  <tr className="bg-surface-container-high/50">
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Probe
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Condition
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Threshold
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Severity
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Status
                    </th>
                    <th className={cn('py-3 px-4 text-right', labelClass)}>
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
                      <td className={cn(!rule.enabled && 'opacity-50', "py-3 px-4 text-sm font-medium text-on-surface")}>
                        {rule.probe_display_name ?? rule.probe_name}
                      </td>
                      <td className={cn(!rule.enabled && 'opacity-50', "py-3 px-4 text-xs text-on-surface-dim uppercase tracking-wider")}>
                        {conditionLabels[rule.condition] ?? rule.condition}
                      </td>
                      <td className={cn('py-3 px-4 text-sm font-mono', isUnconfigured(rule) ? 'text-warning' : 'text-on-surface')}>
                        {formatThreshold(rule)}
                      </td>
                      <td className="py-3 px-4">
                        <Chip tone={rule.severity === 'critical' ? 'critical' : 'warning'}>{rule.severity}</Chip>
                      </td>
                      <td className="py-3 px-4">
                        <Toggle
                      checked={rule.enabled}
                      onChange={() => handleToggle(rule)}
                      label={`${rule.enabled ? 'Disable' : 'Enable'} rule for ${rule.probe_display_name ?? rule.probe_name}`}
                    />
                      </td>
                      <td className="py-3 px-4">
                        <div className="flex items-center justify-end gap-1.5">
                          <IconButton icon={Pencil} label="Edit rule" onClick={() => setEditingRule(rule)} />
                          <ConfirmButton
                      onConfirm={() => deleteMutation.mutate(rule.id)}
                      title="Delete rule"
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
            </Card>
          </>
        )}

      {/* Mutation errors */}
      {(createMutation.isError || updateMutation.isError || deleteMutation.isError) && (
        <div className="bg-tertiary/10 rounded-xl p-3 text-sm text-tertiary">
          {createMutation.isError && `Couldn't create the alert rule: ${createMutation.error.message}. `}
          {updateMutation.isError && `Couldn't update the alert rule: ${updateMutation.error.message}. `}
          {deleteMutation.isError && `Couldn't delete the alert rule: ${deleteMutation.error.message}. `}
        </div>
      )}

      {/* Alert Event Log */}
      <AlertEventLog />

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
      </PageBody>
    </div>
  )
}

function AlertEventLog() {
  const { data, isLoading } = useAlertEvents({ limit: 25 })
  const events = data?.events ?? []

  function formatTime(ts: string) {
    return new Date(ts).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="space-y-3">
      <SectionHeader title="Recent Events" icon={Activity} />

      {isLoading ? (
        <Card flush><LoadingState label="Loading events..." /></Card>
      ) : events.length === 0 ? (
        <Card flush><EmptyState icon={<Activity size={24} />} message="No alert events yet." /></Card>
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {events.map((event) => (
              <div key={event.id} className="bg-surface-container rounded-2xl px-4 py-3 space-y-1.5">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-on-surface">
                    {event.probe_display_name ?? event.probe_name ?? '--'}
                  </span>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <Chip tone={event.severity === 'critical' ? 'critical' : 'warning'}>{event.severity ?? '--'}</Chip>
                    {event.cleared_at ? (
                      <Chip tone="secondary">Cleared</Chip>
                    ) : (
                      <Chip tone="critical" pulse>Active</Chip>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-3 text-xs text-on-surface-dim flex-wrap">
                  <span>Peak: <span className="font-mono text-on-surface">{event.peak_value.toFixed(2)}</span></span>
                  <span>Fired: {formatTime(event.fired_at)}</span>
                  {event.cleared_at && <span>Cleared: {formatTime(event.cleared_at)}</span>}
                </div>
              </div>
            ))}
          </div>
          {/* Desktop table */}
          <Card flush className="hidden sm:block">
              <table className="w-full">
                <thead>
                  <tr className="bg-surface-container-high/50">
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Probe
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Severity
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Peak
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Fired
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Cleared
                    </th>
                    <th className={cn('py-3 px-4 text-left', labelClass)}>
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {events.map((event) => (
                    <tr
                      key={event.id}
                      className="transition-fluid hover:bg-surface-container-high/50"
                    >
                      <td className="py-3 px-4 text-sm font-medium text-on-surface">
                        {event.probe_display_name ?? event.probe_name ?? '--'}
                      </td>
                      <td className="py-3 px-4">
                        <Chip tone={event.severity === 'critical' ? 'critical' : 'warning'}>{event.severity ?? '--'}</Chip>
                      </td>
                      <td className="py-3 px-4 text-sm text-on-surface font-mono">
                        {event.peak_value.toFixed(2)}
                      </td>
                      <td className="py-3 px-4 text-sm text-on-surface-dim">
                        {formatTime(event.fired_at)}
                      </td>
                      <td className="py-3 px-4 text-sm text-on-surface-dim">
                        {event.cleared_at ? formatTime(event.cleared_at) : '--'}
                      </td>
                      <td className="py-3 px-4">
                        {event.cleared_at ? (
                          <Chip tone="secondary">Cleared</Chip>
                        ) : (
                          <Chip tone="critical" pulse>Active</Chip>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          </>
        )}
      </div>
  )
}

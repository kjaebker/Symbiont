import { useState } from 'react'
import { ConfirmButton } from '@/components/ConfirmButton'
import { useSearchParams } from 'react-router-dom'
import { Wrench, Power, Plus, Trash2, CheckCircle2, Pencil } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn, relativeTime } from '@/lib/utils'
import Control from '@/pages/Control'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, EmptyState, Field, Input, PageBody, Select, SectionHeader } from '@/components/ui'
import {
  useMaintenanceTasks,
  useCreateMaintenanceTask,
  useUpdateMaintenanceTask,
  useDeleteMaintenanceTask,
  useCompleteMaintenanceTask,
  useDueItems,
} from '@/hooks/useDosing'
import type { MaintenanceTask, MaintenanceFrequency } from '@/api/client'

const FREQUENCY_LABELS: Record<MaintenanceFrequency, string> = {
  daily: 'Daily',
  every_n_days: 'Every N days',
  weekly: 'Weekly',
  monthly: 'Monthly',
  as_needed: 'As needed',
}

/** "3 days", "2 weeks", "5 months": how long ago a due date passed. */
function overdueBy(iso: string): string {
  const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86400_000)
  if (days < 1) return ''
  if (days < 14) return `${days} day${days === 1 ? '' : 's'}`
  if (days < 60) return `${Math.floor(days / 7)} weeks`
  return `${Math.floor(days / 30)} months`
}

function formatNextDue(iso: string | null): { label: string; overdue: boolean } {
  if (!iso) return { label: 'Not scheduled', overdue: false }
  const d = new Date(iso)
  const now = new Date()
  const diff = d.getTime() - now.getTime()
  if (diff < 0) {
    const late = overdueBy(iso)
    return { label: late ? `Overdue ${late}` : 'Overdue', overdue: true }
  }
  if (diff < 86400_000) return { label: 'Due today', overdue: false }
  return { label: d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }), overdue: false }
}

// ─── Task Form ────────────────────────────────────────────────────────────────

interface TaskFormProps {
  initial?: MaintenanceTask
  onSubmit: (data: {
    name: string; description?: string; frequency: MaintenanceFrequency;
    interval_days?: number; day_of_week?: number; enabled: boolean
  }) => void
  onCancel: () => void
  loading: boolean
}

function TaskForm({ initial, onSubmit, onCancel, loading }: TaskFormProps) {
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [frequency, setFrequency] = useState<MaintenanceFrequency>(initial?.frequency ?? 'weekly')
  const [intervalDays, setIntervalDays] = useState(String(initial?.interval_days ?? ''))
  const [dayOfWeek, setDayOfWeek] = useState(initial?.day_of_week ?? 0)
  const [enabled] = useState(initial?.enabled ?? true)

  return (
    <Card className="space-y-4">
      <Field label="Task name">
        <Input
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="e.g. Water Change"
        />
      </Field>
      <Field label="Description (optional)">
        <Input
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder="Optional details"
        />
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Frequency">
          <Select
            value={frequency}
            onChange={e => setFrequency(e.target.value as MaintenanceFrequency)}
          >
            {(Object.entries(FREQUENCY_LABELS) as [MaintenanceFrequency, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </Select>
        </Field>
        {frequency === 'every_n_days' && (
          <Field label="Every N days">
            <Input
              type="number"
              min="1"
              value={intervalDays}
              onChange={e => setIntervalDays(e.target.value)}
              placeholder="7"
            />
          </Field>
        )}
        {frequency === 'weekly' && (
          <Field label="Day of week">
            <Select
              value={dayOfWeek}
              onChange={e => setDayOfWeek(Number(e.target.value))}
            >
              {['Sunday','Monday','Tuesday','Wednesday','Thursday','Friday','Saturday'].map((d, i) => (
                <option key={i} value={i}>{d}</option>
              ))}
            </Select>
          </Field>
        )}
      </div>
      <div className="flex gap-2 justify-end">
        <Button variant="secondary" onClick={onCancel}>Cancel</Button>
        <Button
          variant="primary"
          disabled={loading || !name}
          onClick={() => onSubmit({
            name,
            description: description || undefined,
            frequency,
            interval_days: frequency === 'every_n_days' && intervalDays ? Number(intervalDays) : undefined,
            day_of_week: frequency === 'weekly' ? dayOfWeek : undefined,
            enabled,
          })}
        >
          {loading ? 'Saving…' : initial ? 'Update' : 'Add Task'}
        </Button>
      </div>
    </Card>
  )
}

// ─── Tasks Tab ────────────────────────────────────────────────────────────────

interface TasksTabProps {
  showForm: boolean
  setShowForm: (v: boolean) => void
}

function TasksTab({ showForm, setShowForm }: TasksTabProps) {
  const { data } = useMaintenanceTasks()
  const tasks = data?.tasks ?? []

  const createTask = useCreateMaintenanceTask()
  const updateTask = useUpdateMaintenanceTask()
  const deleteTask = useDeleteMaintenanceTask()
  const completeTask = useCompleteMaintenanceTask()

  const [editingTask, setEditingTask] = useState<MaintenanceTask | null>(null)
  const [completingId, setCompletingId] = useState<number | null>(null)
  const [noteInputId, setNoteInputId] = useState<number | null>(null)
  const [noteText, setNoteText] = useState('')

  // What counts as "due" is decided by the server (/api/tasks/due), the same
  // list the Dashboard counts, so the two pages can't disagree.
  const { data: dueData } = useDueItems()
  const dueById = new Map(
    (dueData?.items ?? []).filter(i => i.kind === 'task').map(i => [i.id, i]),
  )
  const dueTasks = tasks
    .filter(t => dueById.has(t.id))
    .sort((a, b) => (a.next_due_at ?? '').localeCompare(b.next_due_at ?? ''))

  async function handleComplete(taskId: number) {
    setCompletingId(taskId)
    try {
      await completeTask.mutateAsync({ id: taskId, data: { notes: noteText || undefined } })
      setNoteInputId(null)
      setNoteText('')
    } finally {
      setCompletingId(null)
    }
  }

  return (
    <div className="space-y-6">
      {/* Due now */}
      {dueTasks.length > 0 && (
        <Card className="space-y-4">
          <SectionHeader title="Due Now" />
          <div className="space-y-2">
            {dueTasks.map(t => {
              const overdue = dueById.get(t.id)?.is_overdue ?? false
              const late = overdue && t.next_due_at ? overdueBy(t.next_due_at) : ''
              return (
                <div key={t.id} className={cn(
                  'p-3 rounded-xl space-y-2',
                  overdue ? 'bg-tertiary/10' : 'bg-surface-container-high',
                )}>
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <span className={cn('text-sm font-medium', overdue ? 'text-tertiary' : 'text-on-surface')}>
                        {t.name}
                      </span>
                      {overdue && (
                        <span className="ml-2 text-xs uppercase tracking-wider text-tertiary font-semibold">
                          Overdue{late && ` ${late}`}
                        </span>
                      )}
                    </div>
                    <Button
                      variant="tonal"
                      size="sm"
                      icon={CheckCircle2}
                      className="shrink-0"
                      onClick={() => noteInputId === t.id ? handleComplete(t.id) : setNoteInputId(t.id)}
                      disabled={completingId === t.id}
                    >
                      {completingId === t.id ? 'Logging…' : noteInputId === t.id ? 'Confirm Done' : 'Mark Done'}
                    </Button>
                  </div>
                  {noteInputId === t.id && (
                    <Input
                      autoFocus
                      value={noteText}
                      onChange={e => setNoteText(e.target.value)}
                      placeholder="Optional note (e.g. changed 15 gallons)"
                      onKeyDown={e => e.key === 'Enter' && handleComplete(t.id)}
                    />
                  )}
                </div>
              )
            })}
          </div>
        </Card>
      )}

      {/* All tasks */}
      <Card className="space-y-4">
        <SectionHeader title="Recurring Tasks" />

        {showForm && !editingTask && (
          <div>
            <TaskForm
              onSubmit={async data => {
                await createTask.mutateAsync(data)
                setShowForm(false)
              }}
              onCancel={() => setShowForm(false)}
              loading={createTask.isPending}
            />
          </div>
        )}

        {editingTask && (
          <div>
            <TaskForm
              initial={editingTask}
              onSubmit={async data => {
                await updateTask.mutateAsync({ id: editingTask.id, data })
                setEditingTask(null)
              }}
              onCancel={() => setEditingTask(null)}
              loading={updateTask.isPending}
            />
          </div>
        )}

        {tasks.length === 0 ? (
          <EmptyState message="No tasks yet. Add one above." />
        ) : (
          <div className="space-y-2">
            {tasks.map(t => {
              const { label, overdue } = formatNextDue(t.next_due_at)
              return (
                <div key={t.id} className="p-3 rounded-xl bg-surface-container-high/50 hover:bg-surface-container-high transition-fluid space-y-1.5">
                  {/* Title row */}
                  <div className="flex items-start justify-between gap-2">
                    <span className="text-sm font-medium text-on-surface leading-snug">{t.name}</span>
                    <span className={cn('text-xs font-medium shrink-0 mt-0.5', overdue ? 'text-tertiary' : 'text-on-surface-dim')}>
                      {label}
                    </span>
                  </div>
                  {/* Description + frequency */}
                  <div className="text-xs text-on-surface-dim">
                    {t.description ? `${t.description} · ` : ''}
                    {FREQUENCY_LABELS[t.frequency]}
                    {t.interval_days && t.frequency === 'every_n_days' && ` (${t.interval_days}d)`}
                    {t.last_completed_at && (
                      <span className="text-on-surface-faint"> · Last {relativeTime(t.last_completed_at)}</span>
                    )}
                  </div>
                  {/* Action pills */}
                  <div className="flex gap-1.5 pt-0.5 flex-wrap justify-end">
                    <Button
                      variant="tonal"
                      size="sm"
                      icon={CheckCircle2}
                      onClick={() => handleComplete(t.id)}
                      disabled={completingId === t.id}
                    >
                      {completingId === t.id ? 'Logging…' : 'Done'}
                    </Button>
                    <Button variant="secondary" size="sm" icon={Pencil} onClick={() => setEditingTask(t)}>
                      Edit
                    </Button>
                    <ConfirmButton
                      onConfirm={() => deleteTask.mutate(t.id)}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-surface-container-high text-on-surface-dim hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                    >
                      <Trash2 size={13} />
                      Delete
                    </ConfirmButton>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </Card>
    </div>
  )
}

// ─── Maintenance Page ─────────────────────────────────────────────────────────

export default function MaintenancePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'tasks'
  usePageTitle('Maintenance')

  const [showTaskForm, setShowTaskForm] = useState(false)

  function setTab(t: string) {
    setSearchParams({ tab: t }, { replace: true })
    setShowTaskForm(false)
  }

  return (
    <div>
      <PageHeader
        subtitle="Tank Command"
        title="Maintenance"
        tabs={[
          { key: 'tasks', label: 'Tasks', icon: Wrench },
          { key: 'control', label: 'Control', icon: Power },
        ]}
        activeTab={tab}
        onTabChange={setTab}
        action={tab === 'tasks'
          ? { label: 'Add Task', icon: Plus, onClick: () => setShowTaskForm((v) => !v), active: showTaskForm }
          : null
        }
      />

      {tab === 'tasks' ? (
        <PageBody>
          <TasksTab showForm={showTaskForm} setShowForm={setShowTaskForm} />
        </PageBody>
      ) : (
        <Control hideHeader />
      )}
    </div>
  )
}

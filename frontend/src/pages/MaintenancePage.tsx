import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Wrench, Power, Plus, Trash2, CheckCircle2, Pencil } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn, relativeTime } from '@/lib/utils'
import Control from '@/pages/Control'
import {
  useMaintenanceTasks,
  useCreateMaintenanceTask,
  useUpdateMaintenanceTask,
  useDeleteMaintenanceTask,
  useCompleteMaintenanceTask,
} from '@/hooks/useDosing'
import type { MaintenanceTask, MaintenanceFrequency } from '@/api/client'

const FREQUENCY_LABELS: Record<MaintenanceFrequency, string> = {
  daily: 'Daily',
  every_n_days: 'Every N days',
  weekly: 'Weekly',
  monthly: 'Monthly',
  as_needed: 'As needed',
}

function formatNextDue(iso: string | null): { label: string; overdue: boolean } {
  if (!iso) return { label: 'Not scheduled', overdue: false }
  const d = new Date(iso)
  const now = new Date()
  const diff = d.getTime() - now.getTime()
  if (diff < 0) return { label: 'Overdue', overdue: true }
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
    <div className="bg-surface-container rounded-2xl p-5 space-y-4">
      <div className="space-y-1">
        <label className="text-xs uppercase tracking-wider text-on-surface-dim">Task name</label>
        <input
          className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="e.g. Water Change"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs uppercase tracking-wider text-on-surface-dim">Description (optional)</label>
        <input
          className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder="Optional details"
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Frequency</label>
          <select
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={frequency}
            onChange={e => setFrequency(e.target.value as MaintenanceFrequency)}
          >
            {(Object.entries(FREQUENCY_LABELS) as [MaintenanceFrequency, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
        {frequency === 'every_n_days' && (
          <div className="space-y-1">
            <label className="text-xs uppercase tracking-wider text-on-surface-dim">Every N days</label>
            <input
              type="number"
              min="1"
              className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
              value={intervalDays}
              onChange={e => setIntervalDays(e.target.value)}
              placeholder="7"
            />
          </div>
        )}
        {frequency === 'weekly' && (
          <div className="space-y-1">
            <label className="text-xs uppercase tracking-wider text-on-surface-dim">Day of week</label>
            <select
              className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
              value={dayOfWeek}
              onChange={e => setDayOfWeek(Number(e.target.value))}
            >
              {['Sunday','Monday','Tuesday','Wednesday','Thursday','Friday','Saturday'].map((d, i) => (
                <option key={i} value={i}>{d}</option>
              ))}
            </select>
          </div>
        )}
      </div>
      <div className="flex gap-2 justify-end">
        <button
          onClick={onCancel}
          className="px-4 py-2 rounded-xl text-sm text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid"
        >
          Cancel
        </button>
        <button
          disabled={loading || !name}
          onClick={() => onSubmit({
            name,
            description: description || undefined,
            frequency,
            interval_days: frequency === 'every_n_days' && intervalDays ? Number(intervalDays) : undefined,
            day_of_week: frequency === 'weekly' ? dayOfWeek : undefined,
            enabled,
          })}
          className="px-4 py-2 rounded-xl text-sm font-medium bg-primary/15 text-primary hover:bg-primary/25 disabled:opacity-50 transition-fluid"
        >
          {loading ? 'Saving…' : initial ? 'Update' : 'Add Task'}
        </button>
      </div>
    </div>
  )
}

// ─── Tasks Tab ────────────────────────────────────────────────────────────────

function TasksTab() {
  const { data } = useMaintenanceTasks()
  const tasks = data?.tasks ?? []

  const createTask = useCreateMaintenanceTask()
  const updateTask = useUpdateMaintenanceTask()
  const deleteTask = useDeleteMaintenanceTask()
  const completeTask = useCompleteMaintenanceTask()

  const [showForm, setShowForm] = useState(false)
  const [editingTask, setEditingTask] = useState<MaintenanceTask | null>(null)
  const [completingId, setCompletingId] = useState<number | null>(null)
  const [noteInputId, setNoteInputId] = useState<number | null>(null)
  const [noteText, setNoteText] = useState('')

  const overdueTasks = tasks.filter(t => {
    if (!t.enabled || t.frequency === 'as_needed' || !t.next_due_at) return false
    return new Date(t.next_due_at) <= new Date()
  })
  const upcomingTasks = tasks.filter(t => {
    if (!t.enabled || t.frequency === 'as_needed') return false
    if (!t.next_due_at) return true
    const diff = new Date(t.next_due_at).getTime() - Date.now()
    return diff > 0 && diff <= 86400_000 * 2
  })

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
      {(overdueTasks.length > 0 || upcomingTasks.length > 0) && (
        <div className="bg-surface-container rounded-2xl p-5">
          <h2 className="text-sm uppercase tracking-wider text-on-surface-dim font-semibold mb-4">Due Now</h2>
          <div className="space-y-2">
            {[...overdueTasks, ...upcomingTasks].map(t => {
              const { overdue } = formatNextDue(t.next_due_at)
              return (
                <div key={t.id} className={cn(
                  'p-3 rounded-xl space-y-2',
                  overdue ? 'bg-tertiary/10' : 'bg-surface-container-high',
                )}>
                  <div className="flex items-center justify-between">
                    <div>
                      <span className={cn('text-sm font-medium', overdue ? 'text-tertiary' : 'text-on-surface')}>
                        {t.name}
                      </span>
                      {overdue && (
                        <span className="ml-2 text-xs uppercase tracking-wider text-tertiary font-semibold">Overdue</span>
                      )}
                    </div>
                    <button
                      onClick={() => noteInputId === t.id ? handleComplete(t.id) : setNoteInputId(t.id)}
                      disabled={completingId === t.id}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-primary/15 text-primary hover:bg-primary/25 disabled:opacity-50 transition-fluid"
                    >
                      <CheckCircle2 size={14} />
                      {completingId === t.id ? 'Logging…' : noteInputId === t.id ? 'Confirm Done' : 'Mark Done'}
                    </button>
                  </div>
                  {noteInputId === t.id && (
                    <input
                      autoFocus
                      className="w-full bg-surface-container-high/70 rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
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
        </div>
      )}

      {/* All tasks */}
      <div className="bg-surface-container rounded-2xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm uppercase tracking-wider text-on-surface-dim font-semibold">Recurring Tasks</h2>
          <button
            onClick={() => setShowForm(v => !v)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-primary/10 text-primary hover:bg-primary/20 transition-fluid"
          >
            <Plus size={14} />
            Add Task
          </button>
        </div>

        {showForm && !editingTask && (
          <div className="mb-4">
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
          <div className="mb-4">
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
          <p className="text-sm text-on-surface-dim text-center py-6">
            No tasks yet. Add one above.
          </p>
        ) : (
          <div className="space-y-2">
            {tasks.map(t => {
              const { label, overdue } = formatNextDue(t.next_due_at)
              return (
                <div key={t.id} className="flex items-center gap-3 p-3 rounded-xl bg-surface-container-high/50 hover:bg-surface-container-high transition-fluid">
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-on-surface">{t.name}</div>
                    {t.description && (
                      <div className="text-xs text-on-surface-dim">{t.description}</div>
                    )}
                    <div className="text-xs text-on-surface-dim mt-0.5">
                      {FREQUENCY_LABELS[t.frequency]}
                      {t.interval_days && t.frequency === 'every_n_days' && ` (${t.interval_days}d)`}
                    </div>
                  </div>
                  <div className="text-right shrink-0">
                    <div className={cn('text-xs font-medium', overdue ? 'text-tertiary' : 'text-on-surface-dim')}>
                      {label}
                    </div>
                    {t.last_completed_at && (
                      <div className="text-xs text-on-surface-faint">
                        Last: {relativeTime(t.last_completed_at)}
                      </div>
                    )}
                  </div>
                  <div className="flex gap-1 shrink-0">
                    <button
                      onClick={() => handleComplete(t.id)}
                      disabled={completingId === t.id}
                      className="p-1.5 rounded-lg text-primary hover:bg-primary/10 disabled:opacity-50 transition-fluid"
                      title="Mark done"
                    >
                      <CheckCircle2 size={15} />
                    </button>
                    <button
                      onClick={() => setEditingTask(t)}
                      className="p-1.5 rounded-lg text-on-surface-faint hover:text-primary hover:bg-primary/10 transition-fluid"
                      title="Edit"
                    >
                      <Pencil size={15} />
                    </button>
                    <button
                      onClick={() => deleteTask.mutate(t.id)}
                      className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                      title="Delete"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Maintenance Page ─────────────────────────────────────────────────────────

export default function MaintenancePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'tasks'
  usePageTitle('Maintenance')

  function setTab(t: string) {
    setSearchParams({ tab: t }, { replace: true })
  }

  return (
    <div>
      <div className="px-6 md:px-8 pt-8 max-w-6xl mx-auto">
        <p className="text-xs text-primary uppercase tracking-widest mb-2">Tank Command</p>
        <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight mb-6">Maintenance</h1>
        <div className="flex gap-1.5">
          <button
            onClick={() => setTab('tasks')}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-fluid',
              tab === 'tasks'
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
            )}
          >
            <Wrench size={13} />
            Tasks
          </button>
          <button
            onClick={() => setTab('control')}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-fluid',
              tab === 'control'
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
            )}
          >
            <Power size={13} />
            Control
          </button>
        </div>
      </div>

      {tab === 'tasks' ? (
        <div className="p-6 md:p-8 max-w-6xl mx-auto">
          <TasksTab />
        </div>
      ) : (
        <Control hideHeader />
      )}
    </div>
  )
}

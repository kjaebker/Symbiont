import { useState } from 'react'
import { ScrollText, Plus, Pencil, Trash2, Bot, Cpu, Check, X } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { useJournalEntries, useJournalTemplates, useCreateJournalEntry, useUpdateJournalEntry, useDeleteJournalEntry } from '@/hooks/useJournal'
import { cn, relativeTime } from '@/lib/utils'
import type { JournalEntry, JournalCategory, JournalSentiment, JournalTemplate } from '@/api/client'

// ─── Constants ───────────────────────────────────────────────────────────────

const CATEGORY_LABELS: Record<JournalCategory, string> = {
  observation: 'Observation',
  maintenance: 'Maintenance',
  event: 'Event',
  milestone: 'Milestone',
}

const SENTIMENT_LABELS: Record<JournalSentiment, string> = {
  good: 'Good',
  neutral: 'Neutral',
  bad: 'Bad',
  critical: 'Critical',
}

const SENTIMENT_COLORS: Record<JournalSentiment, string> = {
  good: 'text-secondary bg-secondary/10',
  neutral: 'text-on-surface-dim bg-surface-container-high',
  bad: 'text-tertiary bg-tertiary/10',
  critical: 'text-red-400 bg-red-400/10',
}

const SENTIMENT_DOT: Record<JournalSentiment, string> = {
  good: 'bg-secondary',
  neutral: 'bg-on-surface-faint',
  bad: 'bg-tertiary',
  critical: 'bg-red-400',
}

const CATEGORY_COLORS: Record<JournalCategory, string> = {
  observation: 'text-primary bg-primary/10',
  maintenance: 'text-on-surface-dim bg-surface-container-high',
  event: 'text-violet-400 bg-violet-400/10',
  milestone: 'text-amber-400 bg-amber-400/10',
}

// ─── Entry Form ───────────────────────────────────────────────────────────────

interface EntryFormProps {
  initial?: JournalEntry
  templates: Record<string, JournalTemplate[]>
  onSave: (data: {
    category: JournalCategory
    sentiment?: JournalSentiment | null
    title: string
    body?: string | null
  }) => Promise<void>
  onCancel: () => void
  isSaving: boolean
}

function EntryForm({ initial, templates, onSave, onCancel, isSaving }: EntryFormProps) {
  const [category, setCategory] = useState<JournalCategory>(initial?.category ?? 'observation')
  const [sentiment, setSentiment] = useState<JournalSentiment | ''>(initial?.sentiment ?? '')
  const [title, setTitle] = useState(initial?.title ?? '')
  const [body, setBody] = useState(initial?.body ?? '')
  const [templateOpen, setTemplateOpen] = useState(false)

  function applyTemplate(t: JournalTemplate) {
    setCategory(t.category)
    setSentiment(t.sentiment)
    setTitle(t.title)
    setTemplateOpen(false)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim()) return
    await onSave({
      category,
      sentiment: sentiment || null,
      title: title.trim(),
      body: body.trim() || null,
    })
  }

  const allTemplates = Object.values(templates).flat()

  return (
    <form onSubmit={handleSubmit} className="bg-surface-container-low rounded-2xl p-4 space-y-3">
      {/* Template picker */}
      <div className="relative">
        <button
          type="button"
          onClick={() => setTemplateOpen(v => !v)}
          className="text-xs text-primary/80 hover:text-primary flex items-center gap-1.5 transition-fluid"
        >
          <ScrollText size={12} />
          {templateOpen ? 'Close templates' : 'Use a template'}
        </button>

        {templateOpen && (
          <div className="absolute top-6 left-0 z-20 w-72 bg-surface-container-low rounded-2xl shadow-glow-primary/20 overflow-hidden border border-surface-container-highest/30">
            <div className="max-h-72 overflow-y-auto divide-y divide-surface-container-highest/20">
              {(Object.entries(templates) as [JournalCategory, JournalTemplate[]][]).map(([cat, items]) => (
                <div key={cat}>
                  <p className="px-3 py-1.5 text-xs text-on-surface-faint uppercase tracking-widest font-medium bg-surface-container-highest/20">
                    {CATEGORY_LABELS[cat]}
                  </p>
                  {items.map(t => (
                    <button
                      key={t.title}
                      type="button"
                      onClick={() => applyTemplate(t)}
                      className="w-full text-left flex items-center gap-2.5 px-3 py-2 hover:bg-surface-container-high/60 transition-fluid"
                    >
                      <span className={cn('h-1.5 w-1.5 rounded-full shrink-0', SENTIMENT_DOT[t.sentiment])} />
                      <span className="text-sm text-on-surface">{t.title}</span>
                    </button>
                  ))}
                </div>
              ))}
              {allTemplates.length === 0 && (
                <p className="px-3 py-3 text-sm text-on-surface-faint">No templates loaded.</p>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Category + Sentiment */}
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Category</label>
          <select
            value={category}
            onChange={e => setCategory(e.target.value as JournalCategory)}
            className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
          >
            {(Object.keys(CATEGORY_LABELS) as JournalCategory[]).map(c => (
              <option key={c} value={c}>{CATEGORY_LABELS[c]}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Sentiment</label>
          <select
            value={sentiment}
            onChange={e => setSentiment(e.target.value as JournalSentiment | '')}
            className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
          >
            <option value="">None</option>
            {(Object.keys(SENTIMENT_LABELS) as JournalSentiment[]).map(s => (
              <option key={s} value={s}>{SENTIMENT_LABELS[s]}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Title */}
      <div>
        <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Title *</label>
        <input
          type="text"
          required
          value={title}
          onChange={e => setTitle(e.target.value)}
          placeholder="What happened?"
          className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-sm text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
        />
      </div>

      {/* Body */}
      <div>
        <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Details (optional)</label>
        <textarea
          rows={3}
          value={body}
          onChange={e => setBody(e.target.value)}
          placeholder="Add more context..."
          className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-sm text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid resize-none"
        />
      </div>

      {/* Actions */}
      <div className="flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="px-3 py-1.5 rounded-xl text-sm text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isSaving || !title.trim()}
          className="px-4 py-1.5 rounded-xl bg-primary/10 text-primary text-sm font-medium hover:bg-primary/20 transition-fluid disabled:opacity-50 flex items-center gap-2"
        >
          {isSaving ? 'Saving...' : initial ? <><Check size={14} /> Save</> : <><Plus size={14} /> Add Entry</>}
        </button>
      </div>
    </form>
  )
}

// ─── Entry Card ───────────────────────────────────────────────────────────────

function EntryCard({
  entry,
  templates,
  onEdit,
  onDelete,
}: {
  entry: JournalEntry
  templates: Record<string, JournalTemplate[]>
  onEdit: (e: JournalEntry) => void
  onDelete: (id: number) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const updateMutation = useUpdateJournalEntry()
  const [editing, setEditing] = useState(false)

  async function handleSave(data: Parameters<EntryFormProps['onSave']>[0]) {
    await updateMutation.mutateAsync({ id: entry.id, data })
    setEditing(false)
    onEdit(entry)
  }

  if (editing) {
    return (
      <EntryForm
        initial={entry}
        templates={templates}
        onSave={handleSave}
        onCancel={() => setEditing(false)}
        isSaving={updateMutation.isPending}
      />
    )
  }

  return (
    <div className="bg-surface-container-high/40 rounded-2xl px-4 py-3">
      {/* Header row */}
      <div className="flex items-start gap-3">
        {/* Sentiment dot */}
        {entry.sentiment && (
          <span className={cn('h-2 w-2 rounded-full shrink-0 mt-1.5', SENTIMENT_DOT[entry.sentiment])} />
        )}
        {!entry.sentiment && (
          <span className="h-2 w-2 rounded-full shrink-0 mt-1.5 bg-on-surface-faint/30" />
        )}

        <div className="flex-1 min-w-0">
          {/* Title */}
          <button
            onClick={() => setExpanded(v => !v)}
            className="text-left w-full"
          >
            <p className="text-sm font-semibold text-on-surface leading-snug">{entry.title}</p>
          </button>

          {/* Meta row */}
          <div className="flex flex-wrap items-center gap-1.5 mt-1">
            <span className={cn('text-xs px-1.5 py-0.5 rounded-full font-medium', CATEGORY_COLORS[entry.category])}>
              {CATEGORY_LABELS[entry.category]}
            </span>
            {entry.sentiment && (
              <span className={cn('text-xs px-1.5 py-0.5 rounded-full font-medium', SENTIMENT_COLORS[entry.sentiment])}>
                {SENTIMENT_LABELS[entry.sentiment]}
              </span>
            )}
            {entry.source === 'system' && (
              <span className="text-xs px-1.5 py-0.5 rounded-full font-medium text-on-surface-faint bg-surface-container-highest flex items-center gap-1">
                <Cpu size={10} />
                auto
              </span>
            )}
            {entry.source === 'ai' && (
              <span className="text-xs px-1.5 py-0.5 rounded-full font-medium text-violet-400 bg-violet-400/10 flex items-center gap-1">
                <Bot size={10} />
                AI
              </span>
            )}
            <span className="text-xs text-on-surface-faint ml-auto">{relativeTime(entry.ts)}</span>
          </div>

          {/* Body (expanded) */}
          {expanded && entry.body && (
            <p className="text-sm text-on-surface-dim mt-2 leading-relaxed whitespace-pre-wrap">{entry.body}</p>
          )}
          {!expanded && entry.body && (
            <button onClick={() => setExpanded(true)} className="text-xs text-on-surface-faint hover:text-on-surface-dim mt-1 transition-fluid">
              Show details ↓
            </button>
          )}
          {expanded && entry.body && (
            <button onClick={() => setExpanded(false)} className="text-xs text-on-surface-faint hover:text-on-surface-dim mt-1 transition-fluid block">
              Hide ↑
            </button>
          )}
        </div>

        {/* Actions */}
        {entry.source !== 'system' && (
          <div className="flex items-center gap-1 shrink-0">
            <button
              onClick={() => setEditing(true)}
              className="p-1.5 rounded-lg text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high transition-fluid"
            >
              <Pencil size={13} />
            </button>
            {!deleteConfirm ? (
              <button
                onClick={() => setDeleteConfirm(true)}
                className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
              >
                <Trash2 size={13} />
              </button>
            ) : (
              <div className="flex items-center gap-1">
                <button
                  onClick={() => onDelete(entry.id)}
                  className="px-2 py-1 rounded-lg bg-tertiary/10 text-tertiary text-xs font-medium hover:bg-tertiary/20 transition-fluid"
                >
                  Delete
                </button>
                <button
                  onClick={() => setDeleteConfirm(false)}
                  className="p-1 rounded-lg text-on-surface-faint hover:text-on-surface transition-fluid"
                >
                  <X size={13} />
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Journal() {
  usePageTitle('Journal')

  const [categoryFilter, setCategoryFilter] = useState<JournalCategory | ''>('')
  const [sentimentFilter, setSentimentFilter] = useState<JournalSentiment | ''>('')
  const [showForm, setShowForm] = useState(false)

  const { data, isLoading, isError } = useJournalEntries({
    category: categoryFilter || undefined,
    sentiment: sentimentFilter || undefined,
    limit: 100,
  })
  const { data: templateData } = useJournalTemplates()
  const createMutation = useCreateJournalEntry()
  const deleteMutation = useDeleteJournalEntry()

  const entries = data?.entries ?? []
  const templates = templateData?.templates ?? {}

  async function handleCreate(formData: Parameters<EntryFormProps['onSave']>[0]) {
    await createMutation.mutateAsync({
      category: formData.category,
      sentiment: formData.sentiment ?? undefined,
      title: formData.title,
      body: formData.body ?? undefined,
    })
    setShowForm(false)
  }

  function handleDelete(id: number) {
    deleteMutation.mutate(id)
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">Tank Log</p>
          <h1 className="text-2xl font-bold text-on-surface mt-0.5">Journal</h1>
        </div>
        <button
          onClick={() => setShowForm(v => !v)}
          className={cn(
            'flex items-center gap-2 px-3 py-2 rounded-xl text-sm font-medium transition-fluid',
            showForm
              ? 'bg-surface-container-high text-on-surface'
              : 'bg-primary/10 text-primary hover:bg-primary/20',
          )}
        >
          {showForm ? <X size={14} /> : <Plus size={14} />}
          {showForm ? 'Cancel' : 'New Entry'}
        </button>
      </div>

      {/* New entry form */}
      {showForm && (
        <EntryForm
          templates={templates}
          onSave={handleCreate}
          onCancel={() => setShowForm(false)}
          isSaving={createMutation.isPending}
        />
      )}

      {/* Filters */}
      <div className="space-y-2">
        {/* Category filter */}
        <div className="flex flex-wrap gap-1.5">
          <button
            onClick={() => setCategoryFilter('')}
            className={cn(
              'px-3 py-1 rounded-full text-xs font-medium transition-fluid',
              categoryFilter === ''
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
            )}
          >
            All
          </button>
          {(Object.keys(CATEGORY_LABELS) as JournalCategory[]).map(c => (
            <button
              key={c}
              onClick={() => setCategoryFilter(categoryFilter === c ? '' : c)}
              className={cn(
                'px-3 py-1 rounded-full text-xs font-medium transition-fluid',
                categoryFilter === c
                  ? CATEGORY_COLORS[c]
                  : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
              )}
            >
              {CATEGORY_LABELS[c]}
            </button>
          ))}
        </div>

        {/* Sentiment filter */}
        <div className="flex flex-wrap gap-1.5">
          <button
            onClick={() => setSentimentFilter('')}
            className={cn(
              'px-3 py-1 rounded-full text-xs font-medium transition-fluid',
              sentimentFilter === ''
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
            )}
          >
            Any
          </button>
          {(Object.keys(SENTIMENT_LABELS) as JournalSentiment[]).map(s => (
            <button
              key={s}
              onClick={() => setSentimentFilter(sentimentFilter === s ? '' : s)}
              className={cn(
                'px-3 py-1 rounded-full text-xs font-medium transition-fluid',
                sentimentFilter === s
                  ? SENTIMENT_COLORS[s]
                  : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface',
              )}
            >
              {SENTIMENT_LABELS[s]}
            </button>
          ))}
        </div>
      </div>

      {/* Entry list */}
      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-16 rounded-2xl bg-surface-container-high/40 animate-pulse" />
          ))}
        </div>
      )}

      {isError && (
        <p className="text-sm text-tertiary px-1">Failed to load journal entries.</p>
      )}

      {!isLoading && !isError && entries.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <ScrollText size={36} className="text-on-surface-faint/40" />
          <p className="text-sm text-on-surface-faint">No entries yet.</p>
          {!showForm && (
            <button
              onClick={() => setShowForm(true)}
              className="text-sm text-primary hover:text-primary/80 transition-fluid"
            >
              Add the first entry
            </button>
          )}
        </div>
      )}

      {!isLoading && entries.length > 0 && (
        <div className="space-y-2">
          {entries.map(entry => (
            <EntryCard
              key={entry.id}
              entry={entry}
              templates={templates}
              onEdit={() => {}}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}
    </div>
  )
}

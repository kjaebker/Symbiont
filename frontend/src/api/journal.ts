import { apiFetch } from './_fetch'

export type JournalCategory = 'observation' | 'maintenance' | 'event' | 'milestone'
export type JournalSentiment = 'good' | 'neutral' | 'bad' | 'critical'
export type JournalSource = 'manual' | 'system' | 'ai'

export interface JournalEntry {
  id: number
  ts: string
  category: JournalCategory
  sentiment: JournalSentiment | null
  title: string
  body: string | null
  source: JournalSource
  source_ref: string | null
  created_at: string
}

export interface JournalTemplate {
  category: JournalCategory
  sentiment: JournalSentiment
  title: string
}

export interface JournalListParams {
  category?: JournalCategory
  sentiment?: JournalSentiment
  from?: string
  to?: string
  limit?: number
}

export function getJournalTemplates() {
  return apiFetch<{ templates: Record<JournalCategory, JournalTemplate[]> }>('/api/journal/templates')
}

export function listJournalEntries(params?: JournalListParams) {
  const search = new URLSearchParams()
  if (params?.category) search.set('category', params.category)
  if (params?.sentiment) search.set('sentiment', params.sentiment)
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  if (params?.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiFetch<{ entries: JournalEntry[] }>(`/api/journal${qs ? '?' + qs : ''}`)
}

export function createJournalEntry(data: {
  category: JournalCategory
  sentiment?: JournalSentiment
  title: string
  body?: string
}) {
  return apiFetch<JournalEntry>('/api/journal', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateJournalEntry(id: number, data: {
  category: JournalCategory
  sentiment?: JournalSentiment | null
  title: string
  body?: string | null
}) {
  return apiFetch<JournalEntry>(`/api/journal/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteJournalEntry(id: number) {
  return apiFetch<{ status: string }>(`/api/journal/${id}`, { method: 'DELETE' })
}

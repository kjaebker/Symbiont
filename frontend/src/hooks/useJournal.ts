import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getJournalTemplates,
  listJournalEntries,
  createJournalEntry,
  updateJournalEntry,
  deleteJournalEntry,
} from '@/api/client'
import type { JournalCategory, JournalSentiment, JournalListParams } from '@/api/client'

export function useJournalTemplates() {
  return useQuery({
    queryKey: ['journal-templates'],
    queryFn: getJournalTemplates,
    staleTime: Infinity,
  })
}

export function useJournalEntries(params?: JournalListParams) {
  return useQuery({
    queryKey: ['journal', params],
    queryFn: () => listJournalEntries(params),
    staleTime: 15_000,
  })
}

export function useCreateJournalEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: {
      category: JournalCategory
      sentiment?: JournalSentiment
      title: string
      body?: string
    }) => createJournalEntry(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['journal'] })
    },
  })
}

export function useUpdateJournalEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: {
      id: number
      data: {
        category: JournalCategory
        sentiment?: JournalSentiment | null
        title: string
        body?: string | null
      }
    }) => updateJournalEntry(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['journal'] })
    },
  })
}

export function useDeleteJournalEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteJournalEntry(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['journal'] })
    },
  })
}

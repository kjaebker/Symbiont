import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getLivestock,
  getLivestockSpecies,
  createLivestockItem,
  updateLivestockItem,
  deleteLivestockItem,
  getLivestockObservations,
  createLivestockObservation,
} from '@/api/client'
import type { LivestockItem, LivestockObservation, LivestockType, LivestockStatus } from '@/api/types'

export function useLivestock(params?: { type?: LivestockType; status?: LivestockStatus }) {
  return useQuery({
    queryKey: ['livestock', params],
    queryFn: () => getLivestock(params),
    staleTime: 10_000,
  })
}

export function useLivestockSpecies() {
  return useQuery({
    queryKey: ['livestock-species'],
    queryFn: getLivestockSpecies,
    staleTime: Infinity,
  })
}

export function useCreateLivestockItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (item: Omit<LivestockItem, 'id' | 'created_at' | 'updated_at' | 'image_path'>) =>
      createLivestockItem(item),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
      queryClient.invalidateQueries({ queryKey: ['livestock-species'] })
    },
  })
}

export function useUpdateLivestockItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: number
      data: Partial<Omit<LivestockItem, 'id' | 'created_at' | 'updated_at'>>
    }) => updateLivestockItem(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
      queryClient.invalidateQueries({ queryKey: ['livestock-species'] })
    },
  })
}

export function useDeleteLivestockItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteLivestockItem(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
    },
  })
}

export function useLivestockObservations(livestockId: number, enabled = false) {
  return useQuery({
    queryKey: ['livestock-observations', livestockId],
    queryFn: () => getLivestockObservations(livestockId),
    staleTime: 30_000,
    enabled,
  })
}

export function useCreateLivestockObservation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      livestockId,
      data,
    }: {
      livestockId: number
      data: { status?: LivestockStatus | null; note?: string | null }
    }) => createLivestockObservation(livestockId, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['livestock-observations', variables.livestockId],
      })
      // Invalidate livestock list so status badge refreshes if the handler auto-updated it.
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
    },
  })
}

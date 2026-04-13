import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getLivestock,
  getLivestockItem,
  getLivestockSpecies,
  createLivestockItem,
  updateLivestockItem,
  deleteLivestockItem,
  uploadLivestockImage,
  deleteLivestockImage,
  getLivestockObservations,
  createLivestockObservation,
  uploadObservationImage,
  deleteObservationImage,
} from '@/api/client'
import type { LivestockItem, LivestockType, LivestockStatus } from '@/api/types'

export function useLivestock(params?: { type?: LivestockType; status?: LivestockStatus }) {
  return useQuery({
    queryKey: ['livestock', params],
    queryFn: () => getLivestock(params),
    staleTime: 10_000,
  })
}

export function useLivestockItem(id: number) {
  return useQuery({
    queryKey: ['livestock-item', id],
    queryFn: () => getLivestockItem(id),
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
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
      queryClient.invalidateQueries({ queryKey: ['livestock-item', variables.id] })
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

export function useUploadLivestockImage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) => uploadLivestockImage(id, file),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
      queryClient.invalidateQueries({ queryKey: ['livestock-item', variables.id] })
    },
  })
}

export function useDeleteLivestockImage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteLivestockImage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['livestock'] })
      queryClient.invalidateQueries({ queryKey: ['livestock-item'] })
    },
  })
}

export function useUploadObservationImage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      livestockId,
      obsId,
      file,
    }: {
      livestockId: number
      obsId: number
      file: File
    }) => uploadObservationImage(livestockId, obsId, file),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['livestock-observations', variables.livestockId],
      })
    },
  })
}

export function useDeleteObservationImage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ livestockId, obsId }: { livestockId: number; obsId: number }) =>
      deleteObservationImage(livestockId, obsId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['livestock-observations', variables.livestockId],
      })
    },
  })
}

export function useLivestockObservations(livestockId: number, enabled = true) {
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

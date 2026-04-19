import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getDosingProducts,
  createDosingProduct,
  updateDosingProduct,
  deleteDosingProduct,
  getDosingSchedules,
  createDosingSchedule,
  updateDosingSchedule,
  deleteDosingSchedule,
  logDose,
  getDosingLogs,
  getMaintenanceTasks,
  createMaintenanceTask,
  updateMaintenanceTask,
  deleteMaintenanceTask,
  completeMaintenanceTask,
  getMaintenanceLogs,
  getDueItems,
} from '@/api/client'

// --- Products ---

export function useDosingProducts() {
  return useQuery({
    queryKey: ['dosing-products'],
    queryFn: getDosingProducts,
    staleTime: 60_000,
  })
}

export function useCreateDosingProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createDosingProduct,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dosing-products'] }),
  })
}

export function useUpdateDosingProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateDosingProduct>[1] }) =>
      updateDosingProduct(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dosing-products'] })
      qc.invalidateQueries({ queryKey: ['dosing-schedules'] })
    },
  })
}

export function useDeleteDosingProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDosingProduct(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dosing-products'] })
      qc.invalidateQueries({ queryKey: ['dosing-schedules'] })
    },
  })
}

// --- Schedules ---

export function useDosingSchedules() {
  return useQuery({
    queryKey: ['dosing-schedules'],
    queryFn: getDosingSchedules,
    staleTime: 30_000,
  })
}

export function useCreateDosingSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createDosingSchedule,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dosing-schedules'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useUpdateDosingSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateDosingSchedule>[1] }) =>
      updateDosingSchedule(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dosing-schedules'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useDeleteDosingSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDosingSchedule(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dosing-schedules'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useLogDose() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ scheduleId, data }: { scheduleId: number; data: Parameters<typeof logDose>[1] }) =>
      logDose(scheduleId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['dosing-schedules'] })
      qc.invalidateQueries({ queryKey: ['dosing-logs'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useDosingLogs(params?: Parameters<typeof getDosingLogs>[0]) {
  return useQuery({
    queryKey: ['dosing-logs', params],
    queryFn: () => getDosingLogs(params),
    staleTime: 30_000,
  })
}

// --- Maintenance Tasks ---

export function useMaintenanceTasks() {
  return useQuery({
    queryKey: ['maintenance-tasks'],
    queryFn: getMaintenanceTasks,
    staleTime: 30_000,
  })
}

export function useCreateMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createMaintenanceTask,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['maintenance-tasks'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useUpdateMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateMaintenanceTask>[1] }) =>
      updateMaintenanceTask(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['maintenance-tasks'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useDeleteMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteMaintenanceTask(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['maintenance-tasks'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useCompleteMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data?: Parameters<typeof completeMaintenanceTask>[1] }) =>
      completeMaintenanceTask(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['maintenance-tasks'] })
      qc.invalidateQueries({ queryKey: ['maintenance-logs'] })
      qc.invalidateQueries({ queryKey: ['due-items'] })
    },
  })
}

export function useMaintenanceLogs(taskId: number, limit?: number) {
  return useQuery({
    queryKey: ['maintenance-logs', taskId, limit],
    queryFn: () => getMaintenanceLogs(taskId, limit),
    staleTime: 30_000,
  })
}

// --- Due Items ---

export function useDueItems() {
  return useQuery({
    queryKey: ['due-items'],
    queryFn: getDueItems,
    staleTime: 60_000,
    refetchInterval: 5 * 60_000, // re-check every 5 minutes
  })
}

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
import { qk } from '@/api/queryKeys'

// --- Products ---

export function useDosingProducts() {
  return useQuery({
    queryKey: qk.dosing.products,
    queryFn: getDosingProducts,
    staleTime: 60_000,
  })
}

export function useCreateDosingProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createDosingProduct,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.dosing.products }),
  })
}

export function useUpdateDosingProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateDosingProduct>[1] }) =>
      updateDosingProduct(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.dosing.products })
      qc.invalidateQueries({ queryKey: qk.dosing.schedules })
    },
  })
}

export function useDeleteDosingProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDosingProduct(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.dosing.products })
      qc.invalidateQueries({ queryKey: qk.dosing.schedules })
    },
  })
}

// --- Schedules ---

export function useDosingSchedules() {
  return useQuery({
    queryKey: qk.dosing.schedules,
    queryFn: getDosingSchedules,
    staleTime: 30_000,
  })
}

export function useCreateDosingSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createDosingSchedule,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.dosing.schedules })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useUpdateDosingSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateDosingSchedule>[1] }) =>
      updateDosingSchedule(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.dosing.schedules })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useDeleteDosingSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDosingSchedule(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.dosing.schedules })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useLogDose() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ scheduleId, data }: { scheduleId: number; data: Parameters<typeof logDose>[1] }) =>
      logDose(scheduleId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.dosing.schedules })
      qc.invalidateQueries({ queryKey: qk.dosing.logs() })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useDosingLogs(params?: Parameters<typeof getDosingLogs>[0]) {
  return useQuery({
    queryKey: qk.dosing.logs(params),
    queryFn: () => getDosingLogs(params),
    staleTime: 30_000,
  })
}

// --- Maintenance Tasks ---

export function useMaintenanceTasks() {
  return useQuery({
    queryKey: qk.maintenance.tasks,
    queryFn: getMaintenanceTasks,
    staleTime: 30_000,
  })
}

export function useCreateMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createMaintenanceTask,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.maintenance.tasks })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useUpdateMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateMaintenanceTask>[1] }) =>
      updateMaintenanceTask(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.maintenance.tasks })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useDeleteMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteMaintenanceTask(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.maintenance.tasks })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useCompleteMaintenanceTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data?: Parameters<typeof completeMaintenanceTask>[1] }) =>
      completeMaintenanceTask(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.maintenance.all })
      qc.invalidateQueries({ queryKey: qk.due.all })
    },
  })
}

export function useMaintenanceLogs(taskId: number, limit?: number) {
  return useQuery({
    queryKey: qk.maintenance.logs(taskId, limit),
    queryFn: () => getMaintenanceLogs(taskId, limit),
    staleTime: 30_000,
  })
}

// --- Due Items ---

export function useDueItems() {
  return useQuery({
    queryKey: qk.due.all,
    queryFn: getDueItems,
    staleTime: 60_000,
    refetchInterval: 5 * 60_000, // re-check every 5 minutes
  })
}

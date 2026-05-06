import { apiFetch } from './_fetch'
import type { BackupJob } from './types'

export function getBackups() {
  return apiFetch<{ backups: BackupJob[] }>('/api/system/backups')
}

export function triggerBackup() {
  return apiFetch<BackupJob>('/api/system/backup', { method: 'POST' })
}

export function getBackupConfig() {
  return apiFetch<{ backup_dir: string }>('/api/system/backup/config')
}

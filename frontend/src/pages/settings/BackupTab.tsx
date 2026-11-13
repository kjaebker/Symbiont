import { relativeTime,formatBytes } from '@/lib/utils'
import { Download,RefreshCw } from 'lucide-react'
import { useBackups,useTriggerBackup,useBackupConfig } from '@/hooks/useSettings'
import { Button, Chip, SectionHeader, LoadingState, EmptyState, labelClass } from '@/components/ui'

// =============================================================================
// Backup Tab
// =============================================================================

export default function BackupTab() {
  const { data, isLoading } = useBackups()
  const triggerMutation = useTriggerBackup()
  const { data: configData } = useBackupConfig()

  const backups = data?.backups ?? []

  return (
    <div className="space-y-4">
      {configData?.backup_dir && (
        <div className="mx-4 mt-2 px-4 py-3 bg-surface-container rounded-2xl space-y-1">
          <span className={labelClass}>Backup Location</span>
          <p className="text-xs font-mono text-on-surface-dim break-all">{configData.backup_dir}</p>
          <p className="text-[11px] text-on-surface-faint">
            Both databases are backed up here. Change via <code className="font-mono bg-surface-container-high px-1 rounded">SYMBIONT_BACKUP_DIR</code> in your config.
          </p>
        </div>
      )}

      <SectionHeader
        title="Database Backups"
        className="px-4 pt-2"
        action={
          <Button
            variant="tonal"
            size="sm"
            icon={triggerMutation.isPending ? RefreshCw : Download}
            loading={triggerMutation.isPending}
            onClick={() => triggerMutation.mutate()}
          >
            Run Backup Now
          </Button>
        }
      />

      {triggerMutation.isSuccess && (
        <div className="mx-4 bg-secondary/10 rounded-xl px-4 py-2">
          <p className="text-xs text-secondary font-medium">Backup completed successfully.</p>
        </div>
      )}

      {triggerMutation.isError && (
        <div className="mx-4 bg-tertiary/10 rounded-xl px-4 py-2">
          <p className="text-xs text-tertiary font-medium">Backup failed. Check server logs.</p>
        </div>
      )}

      {isLoading ? (
        <LoadingState label="Loading backups..." />
      ) : backups.length === 0 ? (
        <EmptyState
          icon={<Download size={32} />}
          message="No backups yet. Run a backup to create a snapshot of your databases."
        />
      ) : (
        <>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {backups.map((b) => (
              <div key={b.id} className="bg-surface-container-high rounded-2xl px-4 py-3 space-y-1">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm text-on-surface-dim">{relativeTime(b.ts)}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-mono text-on-surface">{formatBytes(b.size_bytes)}</span>
                    <Chip tone={b.status === 'success' ? 'secondary' : 'critical'}>{b.status}</Chip>
                  </div>
                </div>
                <p className="text-xs text-on-surface-faint font-mono break-all">{b.path}</p>
              </div>
            ))}
          </div>
          {/* Desktop table */}
          <div className="hidden sm:block overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-container-high/50">
                  {['Date', 'Status', 'Size', 'Path'].map((h) => (
                    <th
                      key={h}
                      className={`text-left py-3 px-4 ${labelClass}`}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {backups.map((b) => (
                  <tr key={b.id} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-3 px-4 text-sm text-on-surface-dim">
                      {relativeTime(b.ts)}
                    </td>
                    <td className="py-3 px-4">
                      <Chip tone={b.status === 'success' ? 'secondary' : 'critical'}>{b.status}</Chip>
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface font-mono">
                      {formatBytes(b.size_bytes)}
                    </td>
                    <td className="py-3 px-4 text-sm text-on-surface-dim font-mono truncate max-w-xs">
                      {b.path}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}


import { cn,relativeTime,formatBytes } from '@/lib/utils'
import { Download,RefreshCw } from 'lucide-react'
import { useBackups,useTriggerBackup,useBackupConfig } from '@/hooks/useSettings'
import { LoadingState,EmptyState } from './_shared'

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
          <span className="text-xs text-on-surface-faint uppercase tracking-widest">Backup Location</span>
          <p className="text-xs font-mono text-on-surface-dim break-all">{configData.backup_dir}</p>
          <p className="text-[11px] text-on-surface-faint">
            Both databases are backed up here. Change via <code className="font-mono bg-surface-container-high px-1 rounded">SYMBIONT_BACKUP_DIR</code> in your config.
          </p>
        </div>
      )}

      <div className="flex items-center justify-between px-4 pt-2">
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">
          Database Backups
        </span>
        <button
          onClick={() => triggerMutation.mutate()}
          disabled={triggerMutation.isPending}
          className="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
        >
          {triggerMutation.isPending ? (
            <RefreshCw size={14} className="animate-spin" />
          ) : (
            <Download size={14} />
          )}
          Run Backup Now
        </button>
      </div>

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
                    <span
                      className={cn(
                        'inline-block px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider',
                        b.status === 'success'
                          ? 'bg-secondary/15 text-secondary'
                          : 'bg-tertiary/15 text-tertiary',
                      )}
                    >
                      {b.status}
                    </span>
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
                      className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest"
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
                      <span
                        className={cn(
                          'inline-block px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider',
                          b.status === 'success'
                            ? 'bg-secondary/15 text-secondary'
                            : 'bg-tertiary/15 text-tertiary',
                        )}
                      >
                        {b.status}
                      </span>
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


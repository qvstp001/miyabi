import { LoaderCircleIcon, ScanLineIcon } from 'lucide-react'

import { describeApiError } from '@/api/client'
import { useStartLibraryScan } from '@/api/library'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { notifyTaskError, useNotifyScanTask } from '@/features/tasks/task-toast'

export function LibraryScanButton({
  loading,
  available,
  scanning,
  connected,
  onStarted
}: {
  loading: boolean
  available: boolean
  scanning: boolean
  connected: boolean
  onStarted: () => void
}) {
  const startScan = useStartLibraryScan()
  const notifyScan = useNotifyScanTask()
  // Keep the mutation observer mounted while pagination replaces the visible control.
  if (loading) return <Skeleton className="h-9 w-20 rounded-4xl sm:w-32" />
  if (!available) return null

  const label = scanning ? (connected ? '正在同步' : '连接中') : '同步媒体库'
  const busy = (scanning && connected) || startScan.isPending
  return (
    <div className="flex items-center gap-2">
      <Button
        title="同步文件索引、播放链接和外部 NFO"
        className="w-9 px-0 sm:w-auto sm:px-3"
        disabled={scanning || startScan.isPending}
        onClick={() =>
          startScan.mutate(undefined, {
            onSuccess: task => {
              notifyScan(task)
              onStarted()
            },
            onError: error => {
              notifyTaskError('scan:submit-error', '无法创建同步任务', describeApiError(error))
            }
          })
        }
      >
        {busy ? (
          <LoaderCircleIcon className="size-4 animate-spin" />
        ) : (
          <ScanLineIcon className="size-4" />
        )}
        <span className="hidden sm:inline">{label}</span>
      </Button>
    </div>
  )
}

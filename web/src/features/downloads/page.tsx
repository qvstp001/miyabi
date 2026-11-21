import { Link } from '@tanstack/react-router'

import { downloadStageLabels, useDownloadAction, useDownloads } from '@/api/downloads'
import { AppPage } from '@/components/app-page'
import { ErrorState } from '@/components/error-state'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { ResourceForm } from './resource-form'

export function DownloadsPage() {
  const tasks = useDownloads()
  const retry = useDownloadAction('retry')
  const cancel = useDownloadAction('cancel')
  return <AppPage>
    <PageHeader title="下载任务" description="115 下载 → 重命名和移出文件夹 → LitePan STRM">
      <Button asChild variant="outline"><Link to="/settings">下载设置</Link></Button>
      <Button variant="outline" disabled={tasks.isFetching} onClick={() => void tasks.refetch()}>刷新</Button>
    </PageHeader>
    <Card><CardContent><ResourceForm /></CardContent></Card>
    <p className="text-sm text-muted-foreground">至少 100 MiB 的视频按番号命名，多视频按 cd1、cd2 编号；同名字幕一起移动。仅清理空文件夹，其余附带文件保留原处。</p>
    {tasks.isPending ? <p>正在读取任务…</p> : tasks.isError ? <ErrorState message="下载任务读取失败" onRetry={() => void tasks.refetch()} /> : tasks.data.length === 0 ? <p className="text-muted-foreground">还没有下载任务，可从影片详情选择资源。</p> : tasks.data.map(task => <Card key={task.id}><CardContent className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3"><strong>{task.code}</strong><span className="text-sm">{task.status === 'failed' ? '处理失败，可重试当前阶段' : task.status === 'cancelled' ? '已取消' : downloadStageLabels[task.stage]}</span></div>
      <p className="text-xs text-muted-foreground">115 账号：{task.account_id} · 目录 ID：{task.directory_id}</p>
      <p className="break-all font-mono text-xs text-muted-foreground select-all">{task.uri}</p>
      {task.status === 'running' ? <Progress value={task.progress} /> : null}
      {task.error ? <p className="text-sm text-destructive">{task.error}</p> : null}
      {task.files?.length ? <p className="text-sm break-all">{task.files.map(file => file.name).join('、')}</p> : null}
      <div className="flex gap-2">
        {task.status === 'failed' ? <Button variant="outline" size="sm" disabled={retry.isPending} onClick={() => retry.mutate(task.id)}>重试当前阶段</Button> : null}
        {task.status === 'running' || task.status === 'failed' ? <Button variant="outline" size="sm" disabled={cancel.isPending} onClick={() => cancel.mutate(task.id)}>{task.stage === 'submitting' ? '停止跟踪，请在 115 确认' : '取消任务，保留文件'}</Button> : null}
      </div>
      {task.status === 'done' ? <p className="text-xs text-muted-foreground">LitePan 已接受自动化事件，STRM 任务的执行结果请在 LitePan 中查看。</p> : null}
    </CardContent></Card>)}
  </AppPage>
}
